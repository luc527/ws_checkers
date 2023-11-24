package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

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
	runServer()
}

// TODO mutex around it to allow the server to call them
var webhooks = make(map[string]bool)

func runServer() {
	flag.Parse()

	r := mux.NewRouter()

	r.HandleFunc("/ws", handleWebsocketRequest).Methods("GET")

	r.HandleFunc("/webhook", handleWebhooksRequest).Methods("GET")
	r.HandleFunc("/webhook", registerWebhook).Methods("PUT")
	r.HandleFunc("/webhook", deleteWebhook).Methods("DELETE")

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
        <form hx-put="/webhook" hx-target="#webhooks" hx-swap="innerHTML">
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

func handleWebhooksRequest(w http.ResponseWriter, r *http.Request) {
	if err := webhooksTemplate.ExecuteTemplate(w, "base", webhooks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to write index: %v", err)
	}
}

func registerWebhook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid form: %v", err)
	}
	url := r.Form.Get("url")
	log.Printf("registering webhook with url %v", url)
	webhooks[url] = true
	if err := webhooksTemplate.ExecuteTemplate(w, "table", webhooks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "failed to print table")
	}
}

func deleteWebhook(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	log.Printf("deleting webhook with url %v", url)
	delete(webhooks, url)
	if err := webhooksTemplate.ExecuteTemplate(w, "table", webhooks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to print table: %v", err)
	}
}
