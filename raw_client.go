package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type rawClient struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	stop     chan struct{}
}

func (rc *rawClient) disconnect() {
	select {
	case <-rc.stop:
	default:
		close(rc.stop)
	}
}

func (rc *rawClient) online() bool {
	select {
	case <-rc.stop:
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

func (rc *rawClient) errf(err string, a ...any) {
	err = fmt.Sprintf(err, a...)
	msg := errorMessage(err)
	if bs, err := json.Marshal(msg); err != nil {
		log.Println("raw client: failed to marshal error message")
	} else {
		rc.outgoing <- bs
	}
}

type messageEnvelope struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"data"`
}
