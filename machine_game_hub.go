package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TODO: some way to close the game hub?
// I don't know if it would be necessary

type machineGameHub struct {
	mu    sync.Mutex
	games map[uuid.UUID]*machineGameServer
}

func newMachineGameHub() *machineGameHub {
	return &machineGameHub{
		games: make(map[uuid.UUID]*machineGameServer),
	}
}

func (h *machineGameHub) register(machGame *machineGameServer) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := machGame.id
	if _, exists := h.games[id]; exists {
		return fmt.Errorf("running machine games: uuid conflict")
	}

	h.games[id] = machGame

	go func() {
		timeout := 10 * time.Minute // TODO: either make it a const or let it be customizable (which would be specially helpful for tests)
		timer := time.NewTimer(timeout)
		defer func() {
			timer.Stop()
			h.unregister(id)
		}()
		for {
			select {
			case <-timer.C:
				close(machGame.stop)
				return
			case <-machGame.heartbeat:
				timer.Reset(timeout)
			case <-machGame.ended:
				return
			}
		}
	}()

	return nil
}

func (h *machineGameHub) get(id uuid.UUID) (*machineGameServer, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	server, ok := h.games[id]
	return server, ok
}

func (h *machineGameHub) unregister(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.games, id)
}
