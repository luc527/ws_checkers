package main

import (
	"sync"

	"github.com/google/uuid"
)

type MachineGameHub struct {
	mu    sync.Mutex
	games map[uuid.UUID]*MachineGame
}

func NewMachineGameHub() *MachineGameHub {
	return &MachineGameHub{
		games: make(map[uuid.UUID]*MachineGame),
	}
}

func (h *MachineGameHub) Register(mg *MachineGame) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.games[mg.id] = mg
}

func (h *MachineGameHub) Get(id uuid.UUID) *MachineGame {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.games[id]
}
