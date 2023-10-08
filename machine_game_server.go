package main

import (
	"fmt"
	"log"
	"time"

	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type machineGameServer struct {
	humanColor core.Color
	g          *core.Game
	state      gameState
	v          int
	cli        *gameClient // TODO slice of clients? allow multiple tabs/devices/etc.
	searcher   minimax.Searcher
	humanPlies chan plyRequest
	machPlies  chan core.Ply
	setClient  chan *gameClient
	delClient  chan *gameClient
	ended      chan struct{}
}

func newMachineGameServer(
	humanColor core.Color,
	captureRule core.CaptureRule,
	bestRule core.BestRule,
	timeLimit time.Duration,
	heuristic minimax.Heuristic,
) *machineGameServer {

	if heuristic == nil {
		heuristic = minimax.WeightedCountHeuristic
	}
	searcher := minimax.TimeLimitedSearcher{
		ToMax:     humanColor.Opposite(),
		Heuristic: heuristic,
		TimeLimit: timeLimit,
	}

	return &machineGameServer{
		humanColor: humanColor,
		g:          core.NewGame(captureRule, bestRule),
		v:          1,
		searcher:   searcher,
		humanPlies: make(chan plyRequest),
		machPlies:  make(chan core.Ply),
		setClient:  make(chan *gameClient),
		delClient:  make(chan *gameClient),
	}
}

func (sv *machineGameServer) gameState() gameState {
	if sv.v != sv.state.v {
		sv.state = gameState{
			v:      sv.v,
			board:  *sv.g.Board(),
			toPlay: sv.g.ToPlay(),
			plies:  copyPlies(sv.g.Plies()),
			result: sv.g.Result(),
		}
	}
	log.Printf("game state: %#v\n", sv.state)
	return sv.state
}

func (sv *machineGameServer) runMachineTurn() {
	sv.machPlies <- sv.searcher.Search(sv.g)
}

// TODO timer to stop machine game automatically
// and Reset calls to reset when there's activity

func (sv *machineGameServer) run() {
	g := sv.g

	defer close(sv.ended)

	if g.ToPlay() != sv.humanColor {
		go sv.runMachineTurn()
	}

	for {
		if sv.cli == nil {
			sv.cli = <-sv.setClient
		}
		if sv.cli == nil {
			continue
		}
		cli := sv.cli

		s := sv.gameState()
		cli.gameStates <- s
		if s.result.Over() {
			continue
		}

		select {
		case <-sv.setClient:

		case oldCli := <-sv.delClient:
			if oldCli == sv.cli {
				sv.cli = nil
			}

		case pr := <-sv.humanPlies:
			if g.ToPlay() != sv.humanColor {
				cli.errors <- fmt.Errorf("machine game server: not your turn")
				continue
			}
			plies := g.Plies()
			if pr.v != sv.v || pr.i < 0 || pr.i >= len(plies) {
				cli.errors <- fmt.Errorf("machine game server: stale version or invalid ply")
				continue
			}
			ply := plies[pr.i]
			if _, err := g.DoPly(ply); err != nil {
				cli.errors <- fmt.Errorf("machine game server: %v", err)
				continue
			}
			sv.v++
			cli.gameStates <- sv.gameState()

			if g.Result().Over() {
				return
			} else {
				go sv.runMachineTurn()
			}

		case ply := <-sv.machPlies:
			if g.ToPlay() == sv.humanColor {
				log.Println("machine game server: machine ply attempt on human's turn")
				continue
			}
			if _, err := g.DoPly(ply); err != nil {
				log.Printf("machine game server: machine ply failed: %v\n", err)
				// TODO should this really be done?
				// Try again in a second
				go func() {
					<-time.After(1 * time.Second)
					sv.machPlies <- ply
				}()
				continue
			}
			sv.v++
			cli.gameStates <- sv.gameState()

			if g.Result().Over() {
				return
			}
		}
	}
}
