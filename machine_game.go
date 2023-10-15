package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type MachineGame struct {
	ConcurrentGame
	humanPlayer core.Color
	searcher    minimax.Searcher
}

func NewMachineGame(
	humanPlayer core.Color,
	captureRule core.CaptureRule,
	bestRule core.BestRule,
	heuristic minimax.Heuristic,
	timeLimit time.Duration,
) (*MachineGame, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	searcher := minimax.TimeLimitedSearcher{
		ToMax:     humanPlayer.Opposite(),
		Heuristic: heuristic,
		TimeLimit: timeLimit,
	}
	return &MachineGame{
		ConcurrentGame{
			id:             id,
			g:              core.NewGame(captureRule, bestRule),
			v:              1,
			stateListeners: make(map[chan<- gameState]bool),
		},
		humanPlayer,
		searcher,
	}, nil
}

// Override
func (mg *MachineGame) DoPly(player core.Color, v int, i int) error {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	if player != mg.humanPlayer {
		return fmt.Errorf("machine game: cannot play on behalf of the machine")
	}

	err := mg.doPlyViaId(player, v, i)
	if err != nil {
		return err
	}

	if !mg.gameState().result.Over() {
		copy := mg.g.Copy()
		go func() {
			// If we make a copy of the game, the searcher can do backtracking on it
			// in parallel safely, so it can happen outside of the critical section,
			// which avoids blocking the *ConcurrentGame methods for too long.
			ply := mg.searcher.Search(copy)
			mg.mu.Lock()
			defer mg.mu.Unlock()
			if err := mg.doPly(ply); err != nil {
				log.Printf("error when doing machine ply: %v", err)
			}
		}()
	}

	return nil
}
