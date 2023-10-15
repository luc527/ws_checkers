package main

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

// TODO: check how outgoing and ended can be dealt with in websocket_server.go with this refactoring

type ConcurrentGame struct {
	mu             sync.Mutex
	id             uuid.UUID
	g              *core.Game
	v              int
	state          gameState
	stateListeners map[chan<- gameState]bool
}

func NewConcurrentGame(captureRule core.CaptureRule, bestRule core.BestRule) (*ConcurrentGame, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	g := core.NewGame(captureRule, bestRule)
	return &ConcurrentGame{
		id:             id,
		g:              g,
		v:              1,
		stateListeners: make(map[chan<- gameState]bool),
	}, nil
}

func (cg *ConcurrentGame) gameState() gameState {
	if cg.v != cg.state.v {
		cg.state = gameStateFrom(cg.g, cg.v)
	}
	return cg.state
}

func (cg *ConcurrentGame) Register(c chan<- gameState) {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	state := cg.gameState()
	c <- state
	if state.result.Over() {
		close(c)
	} else {
		cg.stateListeners[c] = true
	}
}

func (cg *ConcurrentGame) unregister(c chan<- gameState) {
	if _, exists := cg.stateListeners[c]; exists {
		delete(cg.stateListeners, c)
		close(c)
	}
}

func (cg *ConcurrentGame) Unregister(c chan<- gameState) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.unregister(c)
}

func (cg *ConcurrentGame) doPly(p core.Ply) error {
	if cg.gameState().result.Over() {
		return fmt.Errorf("can't do ply, game is already over")
	}
	if _, err := cg.g.DoPly(p); err != nil {
		return err
	}
	cg.v++
	state := cg.gameState()
	over := state.result.Over()
	for c := range cg.stateListeners {
		c <- state
		if over {
			cg.unregister(c)
		}
	}
	return nil
}

func (cg *ConcurrentGame) DoPly(player core.Color, v int, i int) error {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	return cg.doPlyViaId(player, v, i)
}

func (cg *ConcurrentGame) doPlyViaId(player core.Color, v int, i int) error {
	if cg.g.Result().Over() {
		return fmt.Errorf("concurrent game: already over")
	}

	if cg.g.ToPlay() != player {
		return fmt.Errorf("concurrent game: not your turn")
	}

	if v != cg.v {
		return fmt.Errorf("concurrent game: stale version")
	}

	plies := cg.g.Plies()
	if i < 0 || i >= len(plies) {
		return fmt.Errorf("concurrent game: ply index out of bounds")
	}

	err := cg.doPly(plies[i])
	if err != nil {
		return err
	}

	return nil
}
