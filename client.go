package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var (
	aiHub = newAIHub()
)

type client struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	stop     chan struct{}
}

func (c *client) close() {
	close(c.stop)
}

func (c *client) trySend(v any) {
	bytes, err := json.Marshal(v)
	log.Printf("trying to send %q, err %v\n", string(bytes), err)
	if err == nil {
		c.outgoing <- bytes
	}
}

func (c *client) sendStr(f string, a ...any) {
	s := fmt.Sprintf(f, a...)
	c.trySend(stringMessage{
		T:    "test",
		Text: s,
	})
}

func (c *client) sendErr(f string, a ...any) {
	err := fmt.Sprintf(f, a...)
	c.trySend(errorMessage(err))
}

func (c *client) handleFirstMessage() {
	timer := time.NewTimer(1 * time.Minute)
	defer timer.Stop()

outer:
	for {
		select {
		case <-timer.C:
			break outer
		case bytes, ok := <-c.incoming:
			if !ok {
				break outer
			}
			envelope := messageEnvelope{}
			if err := json.Unmarshal(bytes, &envelope); err != nil {
				c.sendErr("invalid message format %v", err)
				continue
			}
			switch envelope.T {

			// TODO remove 'against' from new game message
			// then 'namespace' these messages:
			// ai/new, ai/join, ai/reconnect, ...

			case "new-game":
				msg, err := parseNewGameMessage(envelope)
				if err != nil {
					c.sendErr("invalid format: %v", err)
					continue
				}
				c.handleNewGame(msg)
				return
			case "join-game":
				c.sendErr("unimplemented")
				break outer
			case "reconnect-to-game":
				c.sendErr("unimplemented")
				break outer
			default:
				c.sendErr("invalid message type at this point (first message)")
				continue
			}
		}
	}
	c.close()
}

func (c *client) handleNewGame(msg *newGameMessage) {
	req := newRequest[*newGameMessage, token](msg)
	aiHub.register <- req
	select {
	case token := <-req.response:
		c.sendStr("token: %v", token)
	case err := <-req.err:
		c.sendErr("internal")
		log.Printf("new game: %v\n", err)
		c.close()
	}
}
