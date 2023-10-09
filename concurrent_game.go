package main

import (
	"sync"

	"github.com/luc527/go_checkers/core"
)

type concurrentGame struct {
	mu sync.Mutex
	g  *core.Game
	v  int
	s  gameState
}

func newConcurrentGame(g *core.Game) *concurrentGame {
	return &concurrentGame{
		g: g,
		v: 1,
		s: gameState{},
	}
}

func (c *concurrentGame) state() gameState {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.v != c.s.v {
		c.s = gameState{
			v:      c.v,
			board:  *c.g.Board(),
			toPlay: c.g.ToPlay(),
			plies:  copyPlies(c.g.Plies()),
			result: c.g.Result(),
		}
	}
	return c.s
}
