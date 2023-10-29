package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/luc527/go_checkers/conc"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
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

func (c *client) startMachineGame(data machNewData) {
	heuristic := minimax.HeuristicFromString(data.Heuristic)
	if heuristic == nil {
		c.error(fmt.Errorf("unknown heuristic %v", data.Heuristic))
		return
	}

	if data.TimeLimitMs <= 0 {
		c.error(errors.New("non-positive time limit"))
		return
	}
	timeLimit := time.Duration(data.TimeLimitMs * int(time.Millisecond))
	humanColor := data.HumanColor

	mg, err := newMachGame(humanColor, heuristic, timeLimit)
	if err != nil {
		c.error(err)
		return
	}

	mhub.register(mg)

	c.trySend(machConnectedMessageFrom(humanColor, mg.id))
	c.trySend(gameStateMessageFrom(mg.g.CurrentState(), humanColor))

	states := mg.g.NextStates()
	defer mg.g.Detach(states)
	go c.consumeStates(states, humanColor)

	c.runPlayer(humanColor, mg.g)
}

func (c *client) connectToMachineGame(data machConnectData) {
	id := data.Id
	mg, ok := mhub.get(id)
	if !ok {
		c.error(fmt.Errorf("mach/connect: game with id %q not found", id))
		return
	}
	c.trySend(gameStateMessageFrom(mg.g.CurrentState(), mg.humanColor))

	states := mg.g.NextStates()
	defer mg.g.Detach(states)
	go c.consumeStates(states, mg.humanColor)

	c.runPlayer(mg.humanColor, mg.g)
}

func (c *client) startHumanGame(data humanNewData) {
	hg, err := newHumanGame()
	if err != nil {
		c.error(err)
		return
	}

	hhub.register(hg)

	color := data.Color
	opponent := color.Opposite()
	yourToken, oponentToken := hg.tokens[color], hg.tokens[opponent]

	c.trySend(humanCreatedMessageFrom(data.Color, hg.id, yourToken, oponentToken))
	c.trySend(gameStateMessageFrom(hg.g.CurrentState(), color))

	hg.conns.enter(color)

	states := hg.g.NextStates()
	defer hg.g.Detach(states)
	go c.consumeStates(states, color)

	opponentConn := hg.conns.channel(opponent)
	defer hg.conns.detach(opponent, opponentConn)
	go c.consumeConnStates(opponentConn, opponent)

	c.runPlayer(color, hg.g)
}

func (c *client) connectToHumanGame(data humanConnectData) {
	hg, ok := hhub.get(data.Id)
	if !ok {
		c.error(fmt.Errorf("unknown game with id %q", data.Id))
		return
	}

	isWhiteToken := data.Token == hg.tokens[whiteColor]
	isBlackToken := data.Token == hg.tokens[blackColor]

	if !isWhiteToken && !isBlackToken {
		c.error(errors.New("invalid token"))
		return
	}

	var color core.Color
	var token string
	if isWhiteToken {
		color = whiteColor
		token = hg.tokens[whiteColor]
	} else {
		color = blackColor
		token = hg.tokens[blackColor]
	}
	opponent := color.Opposite()

	c.trySend(humanConnectedMessageFrom(color, hg.id, token))
	c.trySend(gameStateMessageFrom(hg.g.CurrentState(), color))

	states := hg.g.NextStates()
	defer hg.g.Detach(states)
	go c.consumeStates(states, color)

	connStates := hg.conns.channel(opponent)
	defer hg.conns.detach(opponent, connStates)
	go c.consumeConnStates(connStates, opponent)

	c.runPlayer(color, hg.g)
}

func (c *client) consumeStates(states <-chan conc.GameState, player core.Color) {
	for s := range states {
		c.trySend(gameStateMessageFrom(s, player))
		if s.Result.Over() {
			close(c.outgoing)
			break
		}
	}
}

func (c *client) consumeConnStates(states <-chan playerConnState, opponent core.Color) {
	for s := range states {
		c.trySend(playerConnStateMessageFrom(s, opponent))
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
