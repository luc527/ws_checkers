package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// TODO: debug hubs to check whether memory is being freed correctly in all instances

var addr = flag.String("addr", ":8080", "http service address")

func runServer() {
	flag.Parse()

	upgrader := websocket.Upgrader{}

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
		if r.URL.Path != "/" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method now allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	server := http.Server{Addr: *addr}

	log.Printf("server running at %v\n", *addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalln(err)
	}
}

func main() {
	runServer()
}
