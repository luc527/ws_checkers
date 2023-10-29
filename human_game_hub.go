package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type humanHub struct {
	mu                sync.Mutex
	games             map[uuid.UUID]*humanGame
	inactivityTimeout time.Duration
}

func newHumanHub(inactivityTimeout time.Duration) *humanHub {
	return &humanHub{
		games:             make(map[uuid.UUID]*humanGame),
		inactivityTimeout: inactivityTimeout,
	}
}

func (h *humanHub) register(hg *humanGame) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.games[hg.id] = hg

	// TODO: when one connection makes the ply that ends the game,
	// the game ends up finishing before we send the final game state to the other player

	// the fix is to only really end the game when both players have disconnected

	states := hg.g.NextStates()
	go func() {
		timer := time.NewTimer(h.inactivityTimeout)
		defer func() {
			h.unregister(hg.id)
			timer.Stop()
			hg.g.DetachAll()
			hg.conns.detachAll()
		}()
		for {
			select {
			case <-timer.C:
				return
			case _, ok := <-states:
				if !ok {
					return
				}
				timer.Reset(h.inactivityTimeout)
			}
		}
	}()
}

func (h *humanHub) unregister(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.games, id)
}

func (h *humanHub) get(id uuid.UUID) (*humanGame, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	hg, ok := h.games[id]
	return hg, ok
}
