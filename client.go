package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var (
	theMachHub = newMachHub()
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

func (c *client) sendStr(t string, f string, a ...any) {
	s := fmt.Sprintf(f, a...)
	c.trySend(stringMessage{
		T:    t,
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
			fmt.Printf("envelope.T: %v\n", envelope.T)
			switch envelope.T {
			case "mach/new":
				msg, err := parseNewMachGameMessage(envelope)
				if err != nil {
					c.sendErr("invalid format: %v", err)
					continue
				}
				c.handleNewMachGame(msg)
				return
			case "mach/join":
				c.sendErr("unimplemented")
				break outer
			case "mach/reconnect":
				c.sendErr("unimplemented")
				break outer
			case "pvp/new":
				c.sendErr("unimplemented")
				break outer
			case "pvp/join":
				c.sendErr("unimplemented")
				break outer
			case "pvp/reconnect":
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

func (c *client) handleNewMachGame(msg *newMachGameMessage) {
	fmt.Printf("handleNewMachGame %v\n", msg)
	req := newRequest[*newMachGameMessage, newMachGameResponse](msg)
	theMachHub.register <- req
	select {
	case res := <-req.response:
		fmt.Printf("res: %v\n", res)
		c.sendStr("test", "token: "+res.token+", id: "+res.id.String())
	case err := <-req.err:
		fmt.Printf("err: %v\n", err)
		c.sendErr("internal")
	}
}
