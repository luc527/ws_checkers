package main

import (
	"encoding/json"

	"github.com/luc527/go_checkers/core"
)

type gameState struct {
	v      int
	board  core.Board
	toPlay core.Color
	plies  []core.Ply
	result core.GameResult
}

type plyRequest struct {
	v int
	i int
}

type gameClient struct {
	plyRequests chan<- plyRequest
	gameStates  chan gameState
	errors      chan error
	*rawClient
}

func newGameClient(plyRequests chan<- plyRequest, raw *rawClient) *gameClient {
	gameStates := make(chan gameState)
	errors := make(chan error)
	return &gameClient{plyRequests, gameStates, errors, raw}
}

func (c *gameClient) run() {
	for {
		select {
		case <-c.stop:
			return
		case err := <-c.errors:
			c.errf("game client: %v", err)
		case s := <-c.gameStates:
			msg := gameStateMessageFrom(s)
			// TODO check if json.Marshal will call the MarshalJSON implementation for core.Ply
			if bs, err := json.Marshal(msg); err != nil {
				c.errf("game client: failed to marshal game state: %v", err)
			} else {
				c.outgoing <- bs
			}
		case bs := <-c.incoming:
			var envelope messageEnvelope
			if err := json.Unmarshal(bs, &envelope); err != nil {
				c.errf("game client: failed to unmarshal envelope: %v", err)
				continue
			}
			// "ply" is the only supported type so far.
			// This could turn into a switch later.
			if envelope.Type != "ply" {
				c.errf("game client: invalid message type, expected 'ply' at this point")
				continue
			}
			var pm plyMessage
			if err := json.Unmarshal(envelope.Raw, &pm); err != nil {
				c.errf("game client: failed to unmarshal ply message: %v", err)
				continue
			}
			c.plyRequests <- plyRequest{v: pm.Version, i: pm.PlyIndex}
		}
	}
}
