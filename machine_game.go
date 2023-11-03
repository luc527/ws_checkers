package main

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

var (
	machMu    = sync.Mutex{}
	machGames = make(map[uuid.UUID]*machGame)
)

type machGame struct {
	id uuid.UUID
	*conGame
	humanColor core.Color
	searcher   minimax.Searcher
}

func newMachGame(searcher minimax.Searcher, humanColor core.Color) (*machGame, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	mg := &machGame{
		id:         id,
		conGame:    newConGame(),
		searcher:   searcher,
		humanColor: humanColor,
	}
	go mg.runMachine()
	return mg, nil
}

func (mg *machGame) doHumanPly(version int, index int) error {
	return mg.conGame.doIndexPly(mg.humanColor, version, index)
}

func (mg *machGame) runMachine() {
	states := mg.conGame.channel()
	machColor := mg.humanColor.Opposite()
	for s := range states {
		if s.toPlay != machColor {
			continue
		}
		if s.result.Over() {
			mg.conGame.detach(states)
			return
		}
		ply := mg.searcher.Search(mg.conGame.game.Copy())
		if err := mg.conGame.doGivenPly(machColor, s.version, ply); err != nil {
			log.Printf("failed to do machine ply: %v", err)
		}
	}
}

func (c *client) startMachineGame(data machNewData) {
	heuristic := minimax.HeuristicFromString(data.Heuristic)
	if heuristic == nil {
		c.errorf("unknown heuristic: %v", heuristic)
		return
	}

	if data.TimeLimitMs <= 0 {
		c.errorf("invalid time (ms) %d", data.TimeLimitMs)
		return
	}
	timeLimit := time.Duration(data.TimeLimitMs * int(time.Millisecond))

	human := data.HumanColor
	searcher := minimax.TimeLimitedSearcher{
		Heuristic: heuristic,
		TimeLimit: timeLimit,
		ToMax:     human.Opposite(),
	}

	mg, err := newMachGame(searcher, human)
	if err != nil {
		c.err(err)
		return
	}

	machMu.Lock()
	machGames[mg.id] = mg
	machMu.Unlock()

	// TODO: Let's leave the timeout as an unimplemented feature
	go func() {
		states := mg.channel()
		for s := range states {
			if s.result.Over() {
				machMu.Lock()
				delete(machGames, mg.id)
				machMu.Unlock()
				mg.detach(states)
			}
		}
	}()

	c.trySend(machConnectedMessageFrom(human, mg.id))
	c.trySend(gameStateMessageFrom(mg.current(), human))

	states := mg.channel()
	go c.consumeGameStates(human, states)

	c.runPlayer(human, mg.conGame)
	mg.detach(states)
}

func (c *client) connectToMachineGame(data machConnectData) {
	machMu.Lock()
	mg := machGames[data.Id]
	machMu.Unlock()

	if mg == nil {
		c.errorf("machine game not found (id %v)", data.Id)
		return
	}

	human := mg.humanColor

	c.trySend(machConnectedMessageFrom(human, mg.id))
	c.trySend(gameStateMessageFrom(mg.current(), human))

	states := mg.channel()
	go c.consumeGameStates(human, states)

	c.runPlayer(human, mg.conGame)
	mg.detach(states)
}
