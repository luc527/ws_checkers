package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// Time to stop and unregister a game with no activity
	heartbeatTimeout = 10 * time.Minute
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

	log.Printf("machine game hub: register %v\n", machGame.id)

	id := machGame.id
	if _, exists := h.games[id]; exists {
		return fmt.Errorf("running machine games: uuid conflict")
	}

	h.games[id] = machGame

	go func() {
		timer := time.NewTimer(heartbeatTimeout)
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
				log.Printf("heartbeat: %v\n", id)
				timer.Reset(heartbeatTimeout)
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
	log.Printf("machine game hub: get %v\n", id)
	server, ok := h.games[id]
	return server, ok
}

func (h *machineGameHub) unregister(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	log.Printf("machine game hub: unregister %v\n", id)
	delete(h.games, id)
}
