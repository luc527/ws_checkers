package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func runServer() {
	flag.Parse()

	server := newWebsocketServer()

	// Just for testing
	server.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("server running at %v\n", *addr)
	go server.serve(*addr)

	for cli := range server.clients {
		cli.handleFirstMessage()
	}
}

func main() {
	runServer()
}
