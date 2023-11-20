package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var port = flag.String("port", "8080", "http service port")

func runServer() {
	flag.Parse()

	// This is not secure, but I'm just trying to avoid cors problems when running on localhost
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("failed to upgrade: %v\n", err)
			return
		}
		cli := websocketRawClient(conn)
		cli.handleFirstMessage()
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello world")
	})

	addr := ":" + *port
	server := http.Server{Addr: addr}

	log.Printf("server running at %v\n", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalln(err)
	}
}

func main() {
	runServer()
}
