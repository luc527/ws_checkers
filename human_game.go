package main

import (
	"github.com/google/uuid"
	"github.com/luc527/go_checkers/conc"
	"github.com/luc527/go_checkers/core"
)

const (
	whiteColor = core.WhiteColor
	blackColor = core.BlackColor
)

type humanGame struct {
	id     uuid.UUID
	g      *conc.Game
	tokens [2]string
	conns  playerConnStates
}

func newHumanGame() (*humanGame, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	tokenForWhite, err := generateToken()
	if err != nil {
		return nil, err
	}
	tokenForBlack, err := generateToken()
	if err != nil {
		return nil, err
	}
	g := conc.NewConcurrentGame()
	hg := &humanGame{
		id: id,
		g:  g,
		tokens: [2]string{
			whiteColor: tokenForWhite,
			blackColor: tokenForBlack,
		},
	}
	hg.conns.init()
	return hg, nil
}
