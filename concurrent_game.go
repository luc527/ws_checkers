package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

type gameState struct {
	board   core.Board
	toPlay  core.Color
	result  core.GameResult
	plies   []core.Ply
	version int
}

type conGame struct {
	gameMu       sync.Mutex
	game         *core.Game
	state        gameState
	lastActivity atomic.Int64

	chansMu sync.Mutex
	chans   map[chan gameState]bool
}

func newConGame() *conGame {
	g := &conGame{
		game:  core.NewGame(),
		chans: make(map[chan gameState]bool),
	}
	g.registerActivity()
	g.updateState()
	return g
}

func (g *conGame) registerActivity() {
	g.lastActivity.Store(time.Now().Unix())
	// log.Println("registering activity", time.Now())
}

func (g *conGame) updateState() {
	g.state.board = *g.game.Board()
	g.state.toPlay = g.game.ToPlay()
	g.state.result = g.game.Result()
	g.state.plies = core.CopyPlies(g.game.Plies())
	g.state.version++
}

func (g *conGame) current() gameState {
	g.gameMu.Lock()
	defer g.gameMu.Unlock()
	return g.state
}

func (g *conGame) channel() chan gameState {
	g.chansMu.Lock()
	defer g.chansMu.Unlock()

	c := make(chan gameState)
	g.chans[c] = true

	return c
}

func (g *conGame) detach(c chan gameState) {
	g.chansMu.Lock()
	defer g.chansMu.Unlock()

	if _, ok := g.chans[c]; ok {
		delete(g.chans, c)
		close(c)
	}
}

func (g *conGame) detachAll() {
	g.chansMu.Lock()
	defer g.chansMu.Unlock()

	for c := range g.chans {
		delete(g.chans, c)
		close(c)
	}
}

func (g *conGame) update(s gameState) {
	g.chansMu.Lock()
	defer g.chansMu.Unlock()

	for c := range g.chans {
		c <- s
	}
}

func (g *conGame) validatePly(player core.Color, version int) error {
	s := g.state
	if s.result.Over() {
		return errors.New("do ply: game already over")
	}
	if version != s.version {
		return errors.New("do ply: stale game state version")
	}
	if s.toPlay != player {
		return errors.New("do ply: not your turn")
	}
	return nil
}

func (g *conGame) doPlyInner(ply core.Ply) error {
	if _, err := g.game.DoPly(ply); err != nil {
		return fmt.Errorf("do ply: %v", err)
	}
	g.updateState()
	g.registerActivity()
	go g.update(g.state)
	return nil
}

func (g *conGame) doGivenPly(player core.Color, version int, ply core.Ply) error {
	g.gameMu.Lock()
	defer g.gameMu.Unlock()
	if err := g.validatePly(player, version); err != nil {
		return err
	}
	if err := g.doPlyInner(ply); err != nil {
		return err
	}
	return nil
}

func (g *conGame) doIndexPly(player core.Color, version int, index int) error {
	g.gameMu.Lock()
	defer g.gameMu.Unlock()
	if err := g.validatePly(player, version); err != nil {
		return err
	}
	n := len(g.state.plies)
	if index < 0 || index >= n {
		return fmt.Errorf("do ply: %d is out of bounds [0, %d)", index, n)
	}
	ply := g.state.plies[index]
	if err := g.doPlyInner(ply); err != nil {
		return err
	}
	return nil
}

func monitorGame[T any](mode string, g *conGame, id uuid.UUID, timeout time.Duration, games map[uuid.UUID]T, mu *sync.Mutex) {
	ticker := time.NewTicker(30 * time.Second)

	go func() {
		states := g.channel()
		for s := range states {
			if s.result.Over() {
				ticker.Stop()
				mu.Lock()
				delete(games, id)
				mu.Unlock()
				g.detach(states)

				go notifyWebhooksGameEnded(mode, id, s)

				break
			}
		}
	}()

	go func() {
		for range ticker.C {
			lastActivity := time.Unix(g.lastActivity.Load(), 0)
			idleDuration := time.Since(lastActivity)
			// log.Printf("game idle for %v (id %v)", idleDuration, id)
			if idleDuration > 2*time.Minute {
				// log.Printf("closing game (id %v)", id)
				g.detachAll()

				mu.Lock()
				delete(games, id)
				mu.Unlock()

				go notifyWebhooksGameEnded(mode, id, g.current())

				break
			}
		}
	}()
}
