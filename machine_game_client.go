package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/luc527/go_checkers/minimax"
)

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

	c.trySend(machConnectedMessageFrom(humanColor, mg.id))
	c.trySend(gameStateMessageFrom(mg.g.CurrentState(), humanColor))

	mg.status.enter()
	gameStates := mg.g.NextStates()

	mhub.register(mg)

	go c.consumeGameStates(data.HumanColor, gameStates)

	defer func() {
		mg.g.Detach(gameStates)
		mg.status.exit()
	}()

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

	mg.status.enter()
	gameStates := mg.g.NextStates()

	go c.consumeGameStates(mg.humanColor, gameStates)

	defer func() {
		mg.g.Detach(gameStates)
		mg.status.exit()
	}()

	c.runPlayer(mg.humanColor, mg.g)
}
