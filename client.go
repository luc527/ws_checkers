package main

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

type rawClient struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	stop     chan struct{}
}

func (c *rawClient) close() {
	close(c.stop)
}

type gameState struct {
	id     uuid.UUID
	board  core.Board
	result core.GameResult
	toPlay core.Color
	plies  []core.Ply
}

type gameClient struct {
	rawClient
	doPly     chan int
	gameState chan gameState
}

func (c *gameClient) run() {
	for {
		select {
		case bytes := <-c.incoming:
			var envelope messageEnvelope
			if err := json.Unmarshal(bytes, &envelope); err != nil {
				// TODO respond with err
				continue
			}
			var ply int
			if err := json.Unmarshal(*envelope.Raw, &ply); err != nil {
				// @CopyPaste!!!!
				// TODO respond with err
				continue
			}
			c.doPly <- ply
		case state := <-c.gameState:
			data := gameStateMessageData{
				GameId: state.id.String(),
				Board:  state.board.Serialize(),
				Result: state.result.String(),
				ToPlay: state.toPlay.String(),
				Plies:  state.plies,
			}
			rawDataBytes, err := json.Marshal(data)
			if err != nil {
				// TODO respond with err
				continue
			}
			rawData := json.RawMessage(rawDataBytes)
			envelope := messageEnvelope{
				T:   "state",
				Raw: &rawData,
			}
			rawEnvelope, err := json.Marshal(envelope)
			if err != nil {
				// TODO respond with err
				continue
			}
			c.outgoing <- rawEnvelope
		}
	}
}
