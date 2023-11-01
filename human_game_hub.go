package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type humanHub struct {
	mu                sync.Mutex
	games             map[uuid.UUID]*humanGame
	inactivityTimeout time.Duration
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

	statesForActivity := hg.g.NextStates()

	go func() {
		var timer *time.Timer

		over := false
		timer = time.NewTimer(h.inactivityTimeout)
		for !over {
			select {
			case <-timer.C:
				over = true
			case _, ok := <-statesForActivity:
				if ok {
					timer.Reset(h.inactivityTimeout)
				} else {
					over = true
				}
			}
		}

		h.unregister(hg.id)
		hg.g.Detach(statesForActivity)
		timer.Stop()

		defer hg.g.DetachAll()

		whiteOnline := hg.statuses[whiteColor].isOnline()
		blackOnline := hg.statuses[blackColor].isOnline()

		if !whiteOnline && !blackOnline {
			return
		}

		whiteStatuses := hg.statuses[whiteColor].channel()
		blackStatuses := hg.statuses[whiteColor].channel()

		defer hg.statuses[whiteColor].detach(whiteStatuses)
		defer hg.statuses[blackColor].detach(blackStatuses)

		timer = time.NewTimer(h.inactivityTimeout)
		for whiteOnline || blackOnline {
			select {
			case <-timer.C:
				// disconnect forcefully
				whiteOnline = false
				blackOnline = false
			case whiteOnline = <-whiteStatuses:
			case blackOnline = <-blackStatuses:
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
