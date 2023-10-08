package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type rawClient struct {
	incoming <-chan []byte
	outgoing chan<- []byte
	stop     chan struct{}
}

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
