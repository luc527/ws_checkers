package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

var (
	machHub = NewMachineGameHub()
)

type client struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	ended    <-chan struct{}
	// TODO: could remove ended from here and let it be a connWriter/connReader implementation detail
}

type gameState struct {
	v      int
	board  core.Board
	result core.GameResult
	toPlay core.Color
	plies  []core.Ply
}

func gameStateFrom(g *core.Game, v int) gameState {
	return gameState{
		v:      v,
		board:  *g.Board(),
		result: g.Result(),
		toPlay: g.ToPlay(),
		plies:  core.CopyPlies(g.Plies()),
	}
}

func (c *client) errf(err string, a ...any) {
	err = fmt.Sprintf(err, a...)
	msg := errorMessage(err)
	if bs, err := json.Marshal(msg); err != nil {
		log.Printf("client: failed to marshal error message: %v", err)
	} else {
		c.outgoing <- bs
	}
}

func (c *client) handleFirstMessage() {
	timer := time.NewTimer(60 * time.Second)
	defer close(c.outgoing)

	for {
		select {
		case <-timer.C:
			c.errf("timeout")
			return
		case bs, ok := <-c.incoming:
			if !ok {
				timer.Stop()
				return
			}
			var envelope messageEnvelope
			if err := json.Unmarshal(bs, &envelope); err != nil {
				c.errf("failed to unmarshal envelope: %v", err)
				continue
			}
			switch envelope.Type {
			case "mach/new":
				var data machNewData
				if err := json.Unmarshal(envelope.Raw, &data); err != nil {
					c.errf("mach/new: failed to unmarshal: %v", err)
					continue
				}
				heuristic := minimax.HeuristicFromString(data.Heuristic)
				if heuristic == nil {
					c.errf("mach/new: unknown heuristic %q", data.Heuristic)
					continue
				}
				if data.TimeLimitMs <= 0 {
					c.errf("mach/new: negative time limit %dms", data.TimeLimitMs)
					continue
				}
				timeLimit := time.Duration(data.TimeLimitMs * int(time.Millisecond))

				mg, err := NewMachineGame(data.HumanColor, data.CapturesMandatory, data.BestMandatory, heuristic, timeLimit)
				if err != nil {
					c.errf("mach/new: failed to create game: %v", err)
					continue
				}

				machId := machIdMessage(mg.id)
				if bs, err := json.Marshal(machId); err != nil {
					c.errf("mach/new: failed to marshal id: %v", err)
					continue
				} else {
					c.outgoing <- bs
				}

				machHub.Register(mg)

				timer.Stop()
				c.playMachineGame(mg)

			case "mach/connect":
				var data machConnectData
				if err := json.Unmarshal(envelope.Raw, &data); err != nil {
					c.errf("mach/connect: failed to unmarshal: %v", err)
					continue
				}
				id := data.Id
				mg := machHub.Get(id)
				if mg == nil {
					c.errf("mach/connect: game not found: id %v", id)
					continue
				}
				c.playMachineGame(mg)
			}
		}
	}
}

func (c *client) playMachineGame(mg *MachineGame) {
	states := make(chan gameState)
	plies := make(chan plyData)
	player := mg.humanPlayer

	go c.consumeStates(states, player)
	go c.producePlies(plies)

	mg.Register(states)
	defer mg.Unregister(states)

	for ply := range plies {
		err := mg.DoPly(player, ply.Version, ply.Index)
		if err != nil {
			c.errf("invalid ply: %v", err)
		}
	}
}

func (c *client) consumeStates(states <-chan gameState, player core.Color) {
	for state := range states {
		bs, err := json.Marshal(gameStateMessageFrom(state, player))
		if err != nil {
			c.errf("failed to marshal game state: %v", err)
			continue
		}
		c.outgoing <- bs
	}
}

func (c *client) producePlies(plies chan<- plyData) {
	defer close(plies)
	for bs := range c.incoming {
		var envelope messageEnvelope
		if err := json.Unmarshal(bs, &envelope); err != nil {
			c.errf("failed to unmarshal envelope for ply: %v", err)
			continue
		}
		var ply plyData
		if err := json.Unmarshal(envelope.Raw, &ply); err != nil {
			c.errf("failed to unmarshal ply: %v", err)
			continue
		}
		plies <- ply
	}
}
