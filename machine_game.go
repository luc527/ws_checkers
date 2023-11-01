package main

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/conc"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type machGame struct {
	id           uuid.UUID
	g            *conc.Game
	searcher     minimax.Searcher
	humanColor   core.Color
	machineColor core.Color
	status       *playerStatus
}

func newMachGame(
	humanColor core.Color,
	heuristic minimax.Heuristic,
	timeLimit time.Duration,
) (*machGame, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	g := conc.NewConcurrentGame()
	machineColor := humanColor.Opposite()
	searcher := minimax.TimeLimitedSearcher{
		ToMax:     machineColor,
		TimeLimit: timeLimit,
		Heuristic: heuristic,
	}
	return &machGame{id, g, searcher, humanColor, machineColor, newPlayerStatus()}, nil
}

func (mg *machGame) runMachine(states <-chan conc.GameState) {
	for s := range states {
		if s.ToPlay != mg.machineColor {
			continue
		}
		mg.runMachineTurn(s.Version)
	}
}

func (mg *machGame) runMachineTurn(v int) {
	ply := mg.searcher.Search(mg.g.UnderlyingGame())
	if err := mg.g.DoPlyGiven(mg.machineColor, v, ply); err != nil {
		log.Printf("error doing machine ply: %v", err)
	}
}
