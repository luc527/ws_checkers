package main

// TODO test both machine and human game with two clients for the same player

import (
	"errors"
	"fmt"

	"github.com/luc527/go_checkers/core"
)

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

	gameStates := hg.g.NextStates()
	opponentStatuses := hg.statuses[opponent].channel()

	c.trySend(humanCreatedMessageFrom(data.Color, hg.id, yourToken, oponentToken))
	c.trySend(gameStateMessageFrom(hg.g.CurrentState(), color))

	go c.consumeGameStates(color, gameStates)
	go c.consumePlayerStatus(opponent, opponentStatuses)
	hg.statuses[color].enter()

	// TODO check order
	// TODO if there's a required order, explain it in a comment
	defer func() {
		hg.g.Detach(gameStates)
		hg.statuses[opponent].detach(opponentStatuses)
		hg.statuses[color].exit()
	}()

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

	// TODO the code below is the same when creating and when connecting
	// should deduplicate it
	// the same for the machine game

	// TODO opponent is online not being sent correctly?

	opponent := color.Opposite()

	gameStates := hg.g.NextStates()
	opponentStatuses := hg.statuses[opponent].channel()

	c.trySend(humanConnectedMessageFrom(color, hg.id, token))
	c.trySend(gameStateMessageFrom(hg.g.CurrentState(), color))
	c.trySend(playerStatusMessageFrom(opponent, hg.statuses[opponent].isOnline()))

	go c.consumeGameStates(color, gameStates)
	go c.consumePlayerStatus(opponent, opponentStatuses)
	hg.statuses[color].enter()

	// TODO check order
	// TODO if there's a required order, explain it in a comment
	defer func() {
		hg.g.Detach(gameStates)
		hg.statuses[opponent].detach(opponentStatuses)
		hg.statuses[color].exit()
	}()

	c.runPlayer(color, hg.g)
}
