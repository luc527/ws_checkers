package main

import (
	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type AIHub struct {
	register chan *request[*newAIGameMessage, token]
}

type AIGame struct {
	g  *core.Game
	ai minimax.Searcher
	id uuid.UUID
	token
}

func newAIHub() *AIHub {
	return &AIHub{
		register: make(chan *request[*newAIGameMessage, token]),
	}
}

func (hub *AIHub) run() {
	for {
		select {
		case req := <-hub.register:
			msg := req.data
			g := core.NewGame(msg.captureRule, msg.bestRule)
			ai := minimax.TimeLimitedSearcher{
				ToMax:     core.BlackColor,
				Heuristic: msg.heuristic,
				TimeLimit: msg.timeLimit,
			}
			token := newToken()
			id, err := uuid.NewRandom()
			if err != nil {
				req.error(err)
			} else {
				_ = &AIGame{g, ai, id, token}
				req.respond(token)
			}
			// TODO: game has a timer
			// when timer ends, game is removed from memory
			// every game action resets the timer
			// the timer is stopped when the game ends
		}
	}
}
