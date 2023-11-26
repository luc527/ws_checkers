package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

const (
	whiteColor = core.WhiteColor
	blackColor = core.BlackColor
)

var (
	humanMu    = sync.Mutex{}
	humanGames = make(map[uuid.UUID]*humanGame)
)

type humanGame struct {
	id uuid.UUID
	*conGame
	tokens [2]string
}

func genToken() (string, error) {
	bs := make([]byte, 36)
	if _, err := rand.Read(bs); err != nil {
		return "", err
	}
	return hex.EncodeToString(bs), nil
}

func newHumanGame() (*humanGame, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	hg := &humanGame{
		id:      id,
		conGame: newConGame(),
		tokens: [2]string{
			whiteColor: "",
			blackColor: "",
		},
	}
	return hg, nil
}

func (c *client) startHumanGame(data humanNewData) {
	color := data.Color

	hg, err := newHumanGame()
	if err != nil {
		c.err(err)
		return
	}

	whiteToken, err := genToken()
	if err != nil {
		c.err(err)
		return
	}
	blackToken, err := genToken()
	if err != nil {
		c.err(err)
		return
	}

	humanMu.Lock()
	humanGames[hg.id] = hg
	humanMu.Unlock()

	go monitorGame(humanMode, hg.conGame, hg.id, 2*time.Minute, humanGames, &humanMu)

	hg.tokens[whiteColor] = whiteToken
	hg.tokens[blackColor] = blackToken

	c.trySend(humanCreatedMessageFrom(color, hg.id, hg.tokens[color], hg.tokens[color.Opposite()]))
	c.trySend(gameStateMessageFrom(hg.conGame.current(), color))

	states := hg.conGame.channel()
	go c.consumeGameStates(color, states)

	c.runPlayer(color, hg.conGame)
	hg.detach(states)
}

func (c *client) connectToHumanGame(data humanConnectData) {
	humanMu.Lock()
	hg := humanGames[data.Id]
	humanMu.Unlock()

	if hg == nil {
		c.errorf("human game not found (id %v)", data.Id)
		return
	}

	var color core.Color
	if data.Token == hg.tokens[whiteColor] {
		color = whiteColor
	} else if data.Token == hg.tokens[blackColor] {
		color = blackColor
	} else {
		c.errorf("invalid token %v", data.Token)
		return
	}

	c.trySend(humanConnectedMessageFrom(color, data.Id, data.Token))
	c.trySend(gameStateMessageFrom(hg.conGame.current(), color))

	states := hg.conGame.channel()
	go c.consumeGameStates(color, states)

	c.runPlayer(color, hg.conGame)
	hg.detach(states)
}
