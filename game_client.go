package main

import (
	"encoding/json"
	"fmt"

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
	raw         *rawClient
	errors      chan error
	gameStates  chan gameState
	plyRequests chan plyRequest
}

// TODO implementar MarshalJSON para core.Ply
// o que vai requerir MarshalJSON para core.Instruction
// fazer no pacote 'core' mesmo
// no final dá pra descomentar o Plies

type gameStateMessage struct {
	Type        string `json:"type"`
	Version     int    `json:"version"`
	Board       string `json:"board"`
	WhiteToPlay bool   `json:"whiteToPlay"`
	Result      string `json:"result"`
	// Plies       []core.Ply `json:"plies"`
}

type plyMessage struct {
	Version  int `json:"version"`
	PlyIndex int `json:"ply"`
}

func gameStateMessageFrom(s gameState) gameStateMessage {
	return gameStateMessage{
		Type:        "state",
		Version:     s.v,
		Board:       s.board.Serialize(),
		WhiteToPlay: s.toPlay == core.WhiteColor,
		Result:      s.result.String(),
	}
}

func plyRequestFrom(pm plyMessage) plyRequest {
	return plyRequest{v: pm.Version, i: pm.PlyIndex}
}

func (c *gameClient) run() {
	for {
		select {
		case <-c.raw.stop:
			return
		case err := <-c.errors:
			c.raw.errf("game client: %v", err)
		case s := <-c.gameStates:
			msg := gameStateMessageFrom(s)
			if bs, err := json.Marshal(msg); err != nil {
				c.raw.errf("game client: failed to marshal game state")
			} else {
				c.raw.outgoing <- bs
			}
		case bs := <-c.raw.incoming:
			var envelope messageEnvelope
			err := json.Unmarshal(bs, &envelope)
			if err != nil {
				c.raw.errf("game client: failed to unmarshal envelope")
				continue
			}
			if envelope.Type != "ply" {
				c.raw.errf("game client: invalid message type (%v)", envelope.Type)
				continue
			}
			var pm plyMessage
			err = json.Unmarshal(envelope.Raw, &pm)
			if err != nil {
				c.raw.errf("game client: failed to unmarshal ply message")
				continue
			}
			pr := plyRequestFrom(pm)
			c.plyRequests <- pr
			fmt.Println("received bytes from client", bs)
		}
	}
}
