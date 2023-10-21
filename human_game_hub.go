package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/conc"
	"github.com/luc527/go_checkers/core"
)

const (
	whiteColor = core.WhiteColor
	blackColor = core.BlackColor
)

type humanHub struct {
	mu                sync.Mutex
	games             map[uuid.UUID]*humanGame
	inactivityTimeout time.Duration
}

type humanGame struct {
	id     uuid.UUID
	g      *conc.Game
	tokens [2]string
}

func newHumanGame(cr core.CaptureRule, br core.BestRule) (*humanGame, error) {
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
	g := conc.NewConcurrentGame(cr, br)
	return &humanGame{
		id, g, [2]string{
			whiteColor: tokenForWhite,
			blackColor: tokenForBlack,
		},
	}, nil
}

func newHumanHub(inactivityTimeout time.Duration) *humanHub {
	return &humanHub{
		games:             make(map[uuid.UUID]*humanGame),
		inactivityTimeout: inactivityTimeout,
	}
}

func (h *humanHub) register(hg *humanGame) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.games[hg.id] = hg

	// TODO: copy-pasted from machine_game_hub; deduplicate?
	// TODO: test
	states := hg.g.NextStates()
	go func() {
		timer := time.NewTimer(h.inactivityTimeout)
		defer func() {
			h.unregister(hg.id)
			timer.Stop()
			hg.g.DetachAll()
		}()
		for {
			select {
			case <-timer.C:
				return
			case _, ok := <-states:
				if !ok {
					return
				}
				timer.Reset(h.inactivityTimeout)
			}
		}
	}()
}

func (h *humanHub) unregister(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.games, id)
}

func (h *humanHub) get(id uuid.UUID) (*humanGame, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	hg, ok := h.games[id]
	return hg, ok
}
