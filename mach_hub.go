package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type machHub struct {
	games    map[uuid.UUID]*machGameHub
	register chan *request[*newMachGameMessage, newMachGameResponse]
	endGame  chan uuid.UUID
}

type machGameHub struct {
	hub   *machHub
	g     *core.Game
	ai    minimax.Searcher
	id    uuid.UUID
	token string
}

type newMachGameResponse struct {
	id    uuid.UUID
	token string
}

func newMachHub() *machHub {
	return &machHub{
		register: make(chan *request[*newMachGameMessage, newMachGameResponse]),
	}
}

func (hub *machHub) run() {
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
				gameHub := &machGameHub{
					hub:   hub,
					g:     g,
					ai:    ai,
					id:    id,
					token: token,
				}
				hub.games[id] = gameHub
				go gameHub.run()
				req.respond(newMachGameResponse{id, token})
			}
		case id := <-hub.endGame:
			delete(hub.games, id)
		}
	}
}

func (h *machGameHub) run() {
	timer := time.NewTimer(10 * time.Minute)
	defer func() {
		h.hub.endGame <- h.id
	}()

	for {
		select {
		case <-timer.C:
			return
		}
	}
}
