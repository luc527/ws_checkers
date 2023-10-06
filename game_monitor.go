package main

import (
	"github.com/luc527/go_checkers/core"
)

type gameState struct {
	v      int
	board  core.Board
	toPlay core.Color
	result core.GameResult
	plies  []core.Ply
}

type plyMessage struct {
	v int
	i int
	c chan bool
}

func (p plyMessage) ret(ok bool) {
	select {
	case p.c <- ok:
		close(p.c)
	default:
	}
}

type gameMonitor struct {
	captureRule core.CaptureRule
	bestRule    core.BestRule
	addListener chan gameStateListener
	delListener chan gameStateListener
	doPly       chan plyMessage
	stopSignal  chan struct{}
}

type gameStateListener chan<- gameState

func newGameMonitor(captureRule core.CaptureRule, bestRule core.BestRule) *gameMonitor {
	return &gameMonitor{
		captureRule: captureRule,
		bestRule:    bestRule,
		addListener: make(chan gameStateListener),
		doPly:       make(chan plyMessage),
		delListener: make(chan gameStateListener),
		stopSignal:  make(chan struct{}),
	}
}

// TODO implement the following two functions in the core package itself

func deepCopyPly(p0 core.Ply) core.Ply {
	p1 := make(core.Ply, len(p0))
	copy(p1, p0)
	return p1
}

func deepCopyPlies(ps0 []core.Ply) []core.Ply {
	ps1 := make([]core.Ply, len(ps0))
	for i, p0 := range ps0 {
		ps1[i] = deepCopyPly(p0)
	}
	return ps1
}

func (m *gameMonitor) run() {
	g := core.NewGame(m.captureRule, m.bestRule)
	v := 1
	listeners := make(map[gameStateListener]bool)

	cachedV := 0
	cachedState := gameState{}

	derive := func() gameState {
		if v == cachedV {
			return cachedState
		}
		cachedV = v
		plies := deepCopyPlies(g.Plies())
		cachedState = gameState{
			v:      v,
			board:  *g.Board(),
			toPlay: g.ToPlay(),
			result: g.Result(),
			plies:  plies,
		}
		return cachedState
	}

	for {
		select {
		case l := <-m.addListener:
			listeners[l] = true
			l <- derive()
		case l := <-m.delListener:
			close(l)
			delete(listeners, l)
		case <-m.stopSignal:
			for l := range listeners {
				close(l)
			}
			return
		case ply := <-m.doPly:
			plies := g.Plies()
			if ply.v != v || g.Result().Over() || ply.i < 0 || ply.i >= len(plies) {
				ply.ret(false)
				continue
			}
			ply.ret(true)
			g.DoPly(plies[ply.i])
			v++
			for l := range listeners {
				select {
				case l <- derive():
				default:
					close(l)
					delete(listeners, l)
				}
			}
		}
	}
}

func (m *gameMonitor) stop() {
	select {
	case <-m.stopSignal:
	default:
		close(m.stopSignal)
	}
}
