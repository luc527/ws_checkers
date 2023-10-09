package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type rawClient struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	stop     chan struct{}
}

// Incoming and outgoing channels are automatically closed
// when stop is closed. This is done in connReader and connWriter.
// (And also in the stdio client below)

func stdioRawClient() *rawClient {
	incoming := make(chan []byte)
	outgoing := make(chan []byte)
	stop := make(chan struct{})

	go func() {
		defer close(incoming)
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			select {
			case incoming <- []byte(in.Text()):
			case <-stop:
				return
			}

		}
		// NOTE: Ignorning potential errors from in.Err()
	}()

	go func() {
		defer close(outgoing)
		for {
			select {
			case bs := <-outgoing:
				fmt.Println(string(bs))
			case <-stop:
				return
			}
		}
	}()

	return &rawClient{incoming, outgoing, stop}
}

func (r *rawClient) disconnect() {
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
}

func (r *rawClient) online() bool {
	select {
	case <-r.stop:
		return false
	default:
		return true
	}
}

type stringMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func errorMessage(err string) stringMessage {
	return stringMessage{
		Type:    "error",
		Message: err,
	}
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
	quit := false
	for !quit {
		select {
		case <-timer.C:
			close(r.stop)
		case bs := <-r.incoming:
			quit = true
			var envelope messageEnvelope
			if err := json.Unmarshal(bs, &envelope); err != nil {
				r.errf("first message: failed to unmarshal envelope")
				continue
			}
			switch envelope.Type {
			case "mach/new": // New vs. machine game
				var msg newMachineGameMessage
				if err := json.Unmarshal(envelope.Raw, &msg); err != nil {
					r.errf("first message: mach/new: failed to unmarshal: %v", err)
					continue
				}

				var heuristic minimax.Heuristic
				switch msg.Heuristic {
				case "UnweightedCount":
					heuristic = minimax.UnweightedCountHeuristic
				case "WeightedCount":
					heuristic = minimax.WeightedCountHeuristic
				default:
					r.errf("first message: mach/new: unknown heuristic %v", msg.Heuristic)
					continue
				}

				captureRule := core.CaptureRule(msg.CapturesMandatory)
				bestRule := core.BestRule(msg.BestMandatory)

				timeLimit := time.Duration(msg.TimeLimitMs * int(time.Millisecond))

				sv := newMachineGameServer(msg.HumanColor, captureRule, bestRule, timeLimit, heuristic)
				go sv.run()

				gc := newGameClient(sv.humanPlies, r)
				go gc.run()

				sv.setClient <- gc

				// TODO: register running game with an id and a token somewhere so the player can reconnect
			}
		}
	}
}
