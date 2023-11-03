package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/luc527/go_checkers/core"
)

type client struct {
	incoming <-chan []byte
	outgoing chan<- []byte
}

func (c *client) error(s string) {
	msg := errorMessage(s)
	if bs, err := json.Marshal(msg); err != nil {
		log.Printf("failed to marshal error: %v", err)
	} else {
		c.outgoing <- bs
	}
}

func (c *client) errorf(f string, v ...any) {
	s := fmt.Sprintf(f, v...)
	c.error(s)
}

func (c *client) err(err error) {
	c.error(err.Error())
}

func (c *client) handleFirstMessage() {
	for bs := range c.incoming {
		var envelope messageEnvelope
		if err := json.Unmarshal(bs, &envelope); err != nil {
			c.err(err)
			return
		}
		switch envelope.Type {
		case "mach/new":
			var data machNewData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.err(err)
				return
			} else {
				c.startMachineGame(data)
			}
		case "mach/connect":
			var data machConnectData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.err(err)
				return
			} else {
				c.connectToMachineGame(data)
			}
		case "human/new":
			var data humanNewData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.err(err)
				return
			} else {
				c.startHumanGame(data)
			}
		case "human/connect":
			var data humanConnectData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.err(err)
				return
			} else {
				c.connectToHumanGame(data)
			}
		default:
			c.errorf("unknown message type %q", envelope.Type)
			return
		}
	}
}

func (c *client) trySend(v any) {
	if bs, err := json.Marshal(v); err != nil {
		c.err(err)
	} else {
		c.outgoing <- bs
	}
}

func (c *client) runPlayer(color core.Color, game *conGame) {
	for bs := range c.incoming {
		var envelope messageEnvelope
		if err := json.Unmarshal(bs, &envelope); err != nil {
			c.err(err)
			continue
		}
		var ply plyData
		if err := json.Unmarshal(envelope.Raw, &ply); err != nil {
			c.err(err)
			continue
		}
		version, index := ply.Version, ply.Index
		if err := game.doIndexPly(color, version, index); err != nil {
			c.err(err)
		}
	}
}

func (c *client) consumeGameStates(player core.Color, states <-chan gameState) {
	for state := range states {
		c.trySend(gameStateMessageFrom(state, player))
		if state.result.Over() {
			close(c.outgoing)
			return
		}
	}
}
