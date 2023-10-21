package main

import (
	"log"
	"sync"
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
}

// TODO: test inactivity timeout

type machHub struct {
	mu                sync.Mutex
	games             map[uuid.UUID]*machGame
	inactivityTimeout time.Duration
}

func newMachGame(
	cr core.CaptureRule,
	br core.BestRule,
	humanColor core.Color,
	heuristic minimax.Heuristic,
	timeLimit time.Duration,
) (*machGame, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	g := conc.NewConcurrentGame(cr, br)
	machineColor := humanColor.Opposite()
	searcher := minimax.TimeLimitedSearcher{
		ToMax:     machineColor,
		TimeLimit: timeLimit,
		Heuristic: heuristic,
	}
	return &machGame{id, g, searcher, humanColor, machineColor}, nil
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

func newMachHub(inactivityTimeout time.Duration) *machHub {
	return &machHub{
		games:             make(map[uuid.UUID]*machGame),
		inactivityTimeout: inactivityTimeout,
	}
}

func (h *machHub) register(mg *machGame) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.games[mg.id] = mg

	s := mg.g.CurrentState()
	if s.ToPlay == mg.machineColor {
		go mg.runMachineTurn(s.Version)
	}

	go mg.runMachine(mg.g.NextStates())

	statesForActivity := mg.g.NextStates()
	go func() {
		timer := time.NewTimer(h.inactivityTimeout)
		defer func() {
			h.unregister(mg.id)
			timer.Stop()
			mg.g.DetachAll()
		}()
		for {
			select {
			case <-timer.C:
				return
			case _, ok := <-statesForActivity:
				if !ok {
					return
				}
				timer.Reset(h.inactivityTimeout)
			}
		}
	}()
}

func (h *machHub) unregister(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.games, id)
}

func (h *machHub) get(id uuid.UUID) (*machGame, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	mg, ok := h.games[id]
	return mg, ok
}
