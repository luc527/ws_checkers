package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var port = flag.String("port", "88", "http service port")

var upgrader = websocket.Upgrader{
	// This is not secure, but I'm just trying to avoid cors problems when running on localhost
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	uuid.SetRand(rand.Reader)

	runServer()
}

func runServer() {
	flag.Parse()

	r := mux.NewRouter()

	r.HandleFunc("/ws", handleWebsocketRequest).Methods("GET")

	r.HandleFunc("/webhook", handleGetWebhooks).Methods("GET")
	r.HandleFunc("/webhook", handlePostWebhook).Methods("POST")
	r.HandleFunc("/webhook", handleDeleteWebhook).Methods("DELETE")

	r.HandleFunc("/games", handleGetGames).Methods("GET")
	r.HandleFunc("/game", handleGetGame).Methods("GET")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello world!")
	}).Methods("GET")

	addr := ":" + *port
	server := http.Server{Addr: addr, Handler: r}

	log.Printf("server running at %v\n", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalln(err)
	}
}
