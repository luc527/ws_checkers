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

func (mg *machGame) runMachine() {
	if !mg.machineHandleState(mg.current()) {
		return
	}

	states := mg.channel()
	for s := range states {
		if !mg.machineHandleState(s) {
			mg.conGame.detach(states)
		}
	}
}

func (mg *machGame) machineHandleState(s gameState) bool {
	machColor := mg.humanColor.Opposite()
	if s.toPlay != machColor {
		return true
	}
	if s.result.Over() {
		return false
	}
	ply := mg.searcher.Search(mg.game.Copy())
	if err := mg.doGivenPly(machColor, s.version, ply); err != nil {
		log.Printf("failed to do machine ply: %v", err)
	}
	return true
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

	go monitorGame("machine", mg.conGame, mg.id, 2*time.Minute, machGames, &machMu)

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
