package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/luc527/go_checkers/conc"
	"github.com/luc527/go_checkers/core"
)

var (
	mhub = newMachHub(10 * time.Minute)
	hhub = newHumanHub(10 * time.Minute)
)

type client struct {
	incoming <-chan []byte
	outgoing chan<- []byte
}

func (c *client) error(err error) {
	msg := errorMessage(err.Error())
	if bs, err := json.Marshal(msg); err != nil {
		log.Printf("failed to marshal error: %v", err)
	} else {
		c.outgoing <- bs
	}
}

func (c *client) handleFirstMessage() {
	for bs := range c.incoming {
		var envelope messageEnvelope
		if err := json.Unmarshal(bs, &envelope); err != nil {
			c.error(err)
			return
		}
		switch envelope.Type {
		case "mach/new":
			var data machNewData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.error(err)
				return
			} else {
				c.startMachineGame(data)
			}
			return
		case "mach/connect":
			var data machConnectData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.error(err)
				return
			} else {
				c.connectToMachineGame(data)
			}
		case "human/new":
			var data humanNewData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.error(err)
				return
			} else {
				c.startHumanGame(data)
			}
		case "human/connect":
			var data humanConnectData
			if err := json.Unmarshal(envelope.Raw, &data); err != nil {
				c.error(err)
				return
			} else {
				c.connectToHumanGame(data)
			}
		}
	}
}

func (c *client) trySend(v any) {
	if bs, err := json.Marshal(v); err != nil {
		c.error(err)
	} else {
		c.outgoing <- bs
	}
}

func (c *client) runPlayer(color core.Color, g *conc.Game) {
	for bs := range c.incoming {
		var envelope messageEnvelope
		if err := json.Unmarshal(bs, &envelope); err != nil {
			c.error(err)
			continue
		}
		var ply plyData
		if err := json.Unmarshal(envelope.Raw, &ply); err != nil {
			c.error(err)
			continue
		}
		version, index := ply.Version, ply.Index
		if err := g.DoPlyIndex(color, version, index); err != nil {
			c.error(err)
		}
	}
}

func (c *client) consumeGameStates(player core.Color, states <-chan conc.GameState) {
	for state := range states {
		c.trySend(gameStateMessageFrom(state, player))
		if state.Result.Over() {
			close(c.outgoing)
			return
		}
	}
}

func (c *client) consumePlayerStatus(player core.Color, status <-chan bool) {
	for online := range status {
		c.trySend(playerStatusMessageFrom(player, online))
	}
}
