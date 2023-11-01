package main

// TODO: same counting technique being used in human game for closing game only when all players have disconnected

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type machHub struct {
	mu                sync.Mutex
	games             map[uuid.UUID]*machGame
	inactivityTimeout time.Duration
}

func newMachHub(inactivityTimeout time.Duration) *machHub {
	return &machHub{
		games:             make(map[uuid.UUID]*machGame),
		inactivityTimeout: inactivityTimeout,
	}
}

func (h *machHub) register(mg *machGame) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.games[mg.id] = mg

	s := mg.g.CurrentState()
	if s.ToPlay == mg.machineColor {
		go mg.runMachineTurn(s.Version)
	}

	go mg.runMachine(mg.g.NextStates())

	statesForActivity := mg.g.NextStates()

	go func() {
		var timer *time.Timer

		// stop game if there's no activity

		timer = time.NewTimer(h.inactivityTimeout)
		over := false
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

		log.Println("game ended (over or timed out)")

		h.unregister(mg.id)
		mg.g.Detach(statesForActivity)
		timer.Stop()

		// wait for player to close its connection,
		// again with a timeout

		defer mg.g.DetachAll()

		if !mg.status.isOnline() {
			log.Println("player already offline, goodbye")
			return
		}
		status := mg.status.channel()

		timer = time.NewTimer(h.inactivityTimeout)
		online := true
		for online {
			select {
			case <-timer.C:
				online = false
			case online = <-status:
			}
		}

		log.Println("finally player offline, goodbye")
	}()
}

func (h *machHub) unregister(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.games, id)
}

func (h *machHub) get(id uuid.UUID) (*machGame, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	mg, ok := h.games[id]
	return mg, ok
}
