package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

type jsonGameState struct {
	Board   core.Board `json:"board"`
	PlyDone core.Ply   `json:"plyDone"`
}

var webhooksTemplate = template.Must(template.New("all").Parse(`
{{define "table"}}
<table id="webhooks-table">
  <tr>
    <td>URL</td>
    <td></td>
  </tr>
  {{range .}}
    <tr>
      <td>{{.}}</td>
      <td>
        <button hx-delete="/webhook?url={{.}}" hx-target="#webhooks-table" hx-swap="outerHTML">
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

func handleWebsocketRequest(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade: %v\n", err)
		return
	}
	cli := websocketRawClient(conn)
	cli.handleFirstMessage()
}

func handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	if urls, err := getWebhooks(db); err != nil {
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
	if urls, err := addWebhook(db, url); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if err := webhooksTemplate.ExecuteTemplate(w, "table", urls); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if urls, err := deleteWebhook(db, url); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if err := webhooksTemplate.ExecuteTemplate(w, "table", urls); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func writeJsonError(w http.ResponseWriter, code int, message string) {
	bytes, err := json.Marshal(struct {
		Message string `json:"message"`
	}{message})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to marshal json error message: %v", err)
		return
	}

	w.WriteHeader(code)
	if _, err := w.Write(bytes); err != nil {
		log.Printf("failed to write json error message: %v", err)
	}
}

func handleGetGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()

	smode := query.Get("mode")
	mode, err := ModeFromString(smode)
	if err != nil {
		writeJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	idString := query.Get("id")
	id, err := uuid.Parse(idString)
	if err != nil {
		writeJsonError(w, http.StatusBadRequest, "invalid uuid")
		return
	}

	plyHistory, err := getPlyHistory(db, mode, id)
	if err != nil {
		writeJsonError(w, http.StatusInternalServerError, "failed to load game history from the database")
		return
	}

	board := new(core.Board)
	core.PlaceInitialPieces(board)

	states := make([]jsonGameState, 0, 1+len(plyHistory))

	for _, ply := range plyHistory {
		states = append(states, jsonGameState{*board, ply})
		core.PerformInstructions(board, ply)
	}
	states = append(states, jsonGameState{*board, nil})

	bytes, err := json.Marshal(states)
	if err != nil {
		writeJsonError(w, http.StatusInternalServerError, "failed to marshal game history")
		return
	}

	if _, err := w.Write(bytes); err != nil {
		writeJsonError(w, http.StatusInternalServerError, "failed to write game history to response body")
	}
}

func handleGetGames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	smode := r.URL.Query().Get("mode")
	mode, err := ModeFromString(smode)
	if err != nil {
		writeJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ids, err := getGameIds(db, mode)
	if err != nil {
		log.Printf("get games failed: %v", err)
		writeJsonError(w, http.StatusInternalServerError, "failed to retrieve game ids")
		return
	}

	bytes, err := json.Marshal(ids)
	if err != nil {
		writeJsonError(w, http.StatusInternalServerError, "json encode failed")
		return
	}

	if _, err := w.Write(bytes); err != nil {
		writeJsonError(w, http.StatusInternalServerError, "response body write failed")
	}
}
