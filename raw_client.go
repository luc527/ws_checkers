package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

var (
	machHub = newMachineGameHub()
)

type rawClient struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	ended    <-chan struct{}
}

func (r *rawClient) errf(err string, a ...any) {
	err = fmt.Sprintf(err, a...)
	msg := errorMessage(err)
	if bs, err := json.Marshal(msg); err != nil {
		log.Println("raw client: failed to marshal error message")
	} else {
		r.outgoing <- bs
	}
}

func (r *rawClient) handleFirstMessage() {
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			close(r.outgoing)
			return
		case bs := <-r.incoming:
			var envelope messageEnvelope
			if err := json.Unmarshal(bs, &envelope); err != nil {
				r.errf("mach/new: failed to unmarshal envelope")
				continue
			}
			switch envelope.Type {
			case "mach/new": // New vs. machine game
				var msg newMachineGameMessage
				if err := json.Unmarshal(envelope.Raw, &msg); err != nil {
					r.errf("mach/new: failed to unmarshal: %v", err)
					continue
				}
				if r.handleNewMachineGame(msg) {
					return
				}
			case "mach/reconnect":
				var msg reconnectMachineGameMessage
				if err := json.Unmarshal(envelope.Raw, &msg); err != nil {
					r.errf("mach/reconnect: failed to unmarshal: %v", err)
					continue
				}
				if r.handleMachineGameReconnect(msg) {
					return
				}
			}
		}
	}
}

func (r *rawClient) handleNewMachineGame(msg newMachineGameMessage) bool {
	// TODO: heuristic marshalJSON unmarshalJSON
	var heuristic minimax.Heuristic
	switch msg.Heuristic {
	case "UnweightedCount":
		heuristic = minimax.UnweightedCountHeuristic
	case "WeightedCount":
		heuristic = minimax.WeightedCountHeuristic
	default:
		r.errf("mach/new: mach/new: unknown heuristic %v", msg.Heuristic)
		return false
	}

	captureRule := core.CaptureRule(msg.CapturesMandatory)
	bestRule := core.BestRule(msg.BestMandatory)

	timeLimit := time.Duration(msg.TimeLimitMs * int(time.Millisecond))

	sv, err := newMachineGameServer(msg.HumanColor, captureRule, bestRule, timeLimit, heuristic)
	if err != nil {
		log.Printf("machine game server: %v\n", err)
		r.errf("mach/new: couldn't start game server")
		return false
	}

	// TODO: test id being returned

	idMsg, err := json.Marshal(gameIdMessageFrom(sv.id))
	if err != nil {
		log.Printf("game id message: %v\n", err)
		r.errf("mach/new: couldn't marshall game id message")
		return false
	}

	// The code below should only be executed if everything else succeeded

	// TODO: for testing, remove later
	go func() {
		<-sv.ended
		log.Printf("ended: %v\n", sv.id)
	}()

	if err := machHub.register(sv); err != nil {
		r.errf("mach/new: failed to register, %v", err)
		return false
	}

	select {
	case <-r.ended:
		machHub.unregister(sv.id)
		return false
	case r.outgoing <- idMsg:
	}

	go sv.run()

	gc := newGameClient(sv.humanPlies, r)
	go gc.run()

	sv.client <- gc

	return true
}

func (r *rawClient) handleMachineGameReconnect(msg reconnectMachineGameMessage) bool {
	id, err := uuid.Parse(msg.Id)
	if err != nil {
		r.errf("mach/reconnect: invalid id %v", msg.Id)
		return false
	}

	sv, ok := machHub.get(id)
	if !ok {
		r.errf("mach/reconnect: unknown id %v", id)
		return false
	}

	// TODO: this is being done to signal to the client that the reconenction attempt has been successful
	// but maybe it should be a different message type
	idMsg, err := json.Marshal(gameIdMessageFrom(sv.id))
	if err != nil {
		log.Printf("game id message: %v\n", err)
		r.errf("mach/reconnect: couldn't marshall game id message")
		return false
	}

	select {
	case <-r.ended:
		return false
	case r.outgoing <- idMsg:
	}

	gc := newGameClient(sv.humanPlies, r)
	go gc.run()

	sv.client <- gc

	return true
}
