package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type machineGameServer struct {
	id         uuid.UUID
	g          *core.Game
	humanColor core.Color
	state      gameState
	v          int
	cli        *gameClient // TODO slice of clients? allow multiple tabs/devices/etc.
	searcher   minimax.Searcher
	humanPlies chan plyRequest
	machPlies  chan core.Ply
	client     chan *gameClient

	stop      chan struct{}
	ended     chan struct{}
	heartbeat chan struct{}
}

func newMachineGameServer(
	humanColor core.Color,
	captureRule core.CaptureRule,
	bestRule core.BestRule,
	timeLimit time.Duration,
	heuristic minimax.Heuristic,
) (*machineGameServer, error) {

	if heuristic == nil {
		heuristic = minimax.WeightedCountHeuristic
	}
	searcher := minimax.TimeLimitedSearcher{
		ToMax:     humanColor.Opposite(),
		Heuristic: heuristic,
		TimeLimit: timeLimit,
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// TODO: when the game ends, what to do with all those channels
	// need to avoid goroutine leak
	// every place that sends to one of those channels needs to check first if the game has ended?

	return &machineGameServer{
		id:         id,
		humanColor: humanColor,
		g:          core.NewGame(captureRule, bestRule),
		v:          1,
		searcher:   searcher,
		humanPlies: make(chan plyRequest), // passed to game clients so they can send plies to the server
		machPlies:  make(chan core.Ply),   // used by `go sv.runMachineTurn()` to send the machine's ply choice
		client:     make(chan *gameClient),
		stop:       make(chan struct{}),    // closed from outside when the someone wants to stop the game
		ended:      make(chan struct{}),    // closed from the inside to signal that the game server has ended (not necessarily that the game itself is over)
		heartbeat:  make(chan struct{}, 8), // struct{}{}s sent from the inside to signal that the game still has activity
	}, nil
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
	return sv.state
}

func (sv *machineGameServer) runMachineTurn() {
	// Since this is concurrent with (*machineGameServer).run(),
	// it needs to create a copy of the game. (*core.Game).Copy()
	// deep-copies board, which is what we want, and shallow-copies
	// plies, with which there's no problem since only
	// (*machineGameServer).run() changes it.

	sv.machPlies <- sv.searcher.Search(sv.g.Copy())
}

func (sv *machineGameServer) run() {
	defer func() {
		close(sv.ended)
		close(sv.heartbeat)
	}()

	g := sv.g

	if g.ToPlay() != sv.humanColor {
		go sv.runMachineTurn()
	}

	for {
		if sv.cli == nil {
			var cli *gameClient

			select {
			case c := <-sv.client:
				sv.heartbeat <- struct{}{}
				cli = c
			case <-sv.stop:
				return
			}

			if cli != nil {
				sv.cli = cli
				log.Println("machine game server: got client")
				s := sv.gameState()
				cli.gameStates <- s
			} else {
				continue
			}
		}
		cli := sv.cli

		select {
		case <-sv.stop:
			return

		case <-cli.ended:
			log.Println("machine game server: client ended")
			sv.heartbeat <- struct{}{}
			sv.cli = nil

		case <-sv.client:
			sv.heartbeat <- struct{}{}

		case pr := <-sv.humanPlies:
			sv.heartbeat <- struct{}{}
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
			sv.heartbeat <- struct{}{}
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
					log.Printf("machine game server: trying again after 1 second\n")
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
