package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var port = flag.String("port", "8088", "http service port")

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

func handleWebsocketRequest(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade: %v\n", err)
		return
	}
	cli := websocketRawClient(conn)
	cli.handleFirstMessage()
}

// TODO persist webhooks in boltdb

var webhooksTemplate = template.Must(template.New("all").Parse(`
{{define "table"}}
<table id="webhooks-table">
  <tr>
    <td>URL</td>
    <td></td>
  </tr>
  {{range $k, $v := .}}
    <tr>
      <td>{{$k}}</td>
      <td>
        <button hx-delete="/webhook?url={{$k}}" hx-target="#webhooks-table" hx-swap="outerHTML">
          Delete
        </button>
      </td>
    </tr>
  {{else}}
    <tr>
      <td colspan=2>No webhooks have been registered</td>
    </tr>
  {{end}}
</table>
{{end}}

{{define "base"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Checkers Webhooks</title>
  <style>
    * {
      font-family: sans-serif;
    }
    code, pre {
      font-family: monospace;
    }
  </style>
</head>
<body>
  <script src="https://unpkg.com/htmx.org@1.9.9"></script>
  <div style="width: 70%; margin: auto">
    <div style="display: flex; flex-diretion: row">
      <div style="flex: 1">
        <form hx-post="/webhook" hx-target="#webhooks" hx-swap="innerHTML">
          <p>Add Webhook</p>
          <label>URL</label>
          <input name="url" type="url" />
          <button type="submit">Add</button>
        </form>
      </div>
      <div id="webhooks" style="flex: 1">
        {{template "table" .}}
      </div>
    </div>
  </div>
</body>
</html>
{{end}}
`))

func handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	if urls, err := getWebhooks(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if err := webhooksTemplate.ExecuteTemplate(w, "base", urls); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handlePostWebhook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	url := r.Form.Get("url")
	if err := addWebhook(url); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := webhooksTemplate.ExecuteTemplate(w, "table", webhooks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if err := deleteWebhook(url); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := webhooksTemplate.ExecuteTemplate(w, "table", webhooks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
