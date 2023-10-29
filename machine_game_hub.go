package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type machHub struct {
	mu                sync.Mutex
	games             map[uuid.UUID]*machGame
	inactivityTimeout time.Duration
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
