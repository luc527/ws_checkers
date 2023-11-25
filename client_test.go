package main

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/luc527/go_checkers/core"
)

func dumbClient() (*client, chan []byte, chan []byte) {
	incoming := make(chan []byte)
	outgoing := make(chan []byte)
	return &client{incoming, outgoing}, incoming, outgoing
}

func tryJson(t *testing.T, m map[string]any) []byte {
	bs, err := json.Marshal(m)
	if err != nil {
		t.Logf("sendJson: error marshalling: %v", err)
		t.FailNow()
	}
	return bs
}

func trySend(t *testing.T, conn *websocket.Conn, data []byte) {
	err := conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Logf("failed to write message to websocket: %v", err)
		t.FailNow()
	}
}

func tryRead(t *testing.T, conn *websocket.Conn) map[string]any {
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Logf("failed to read message from websocket: %v", err)
		t.FailNow()
	}
	m := make(map[string]any)
	if err := json.Unmarshal(data, &m); err != nil {
		t.Logf("failed to unmarshal message from websocket: %v", err)
		t.FailNow()
	}
	return m
}

func tryGet(t *testing.T, m map[string]any, k string) any {
	v, ok := m[k]
	if !ok {
		t.Logf("map has no key %v", k)
		t.FailNow()
	}
	return v
}

func tryState(t *testing.T, m map[string]any) gameStateMessage {
	typ := tryType(t, "state", tryGet(t, m, "type").(string))

	board := tryGet(t, m, "board").(string)
	b := new(core.Board)
	if err := b.Unserialize([]byte(board)); err != nil {
		t.Logf("invalid board %q", board)
		t.FailNow()
	}

	version := int(tryGet(t, m, "version").(float64))

	result := tryGet(t, m, "result").(string)
	r := new(core.GameResult)
	if err := r.UnmarshalJSON([]byte("\"" + result + "\"")); err != nil {
		t.Logf("invalid result %q", result)
		t.FailNow()
	}

	var plies []core.Ply
	stringPlies := tryGet(t, m, "plies").([]any)
	for _, anyPly := range stringPlies {
		stringPly := anyPly.(string)
		var ply core.Ply
		if err := ply.UnmarshalJSON([]byte("\"" + stringPly + "\"")); err != nil {
			t.Logf("failed to unmarshal ply: %v", err)
			t.FailNow()
		}
		plies = append(plies, ply)
	}

	var toPlay core.Color
	var yourColor core.Color

	if err := toPlay.UnmarshalJSON([]byte("\"" + tryGet(t, m, "toPlay").(string) + "\"")); err != nil {
		t.Logf("invalid toPlay")
		t.FailNow()
	}

	if err := yourColor.UnmarshalJSON([]byte("\"" + tryGet(t, m, "yourColor").(string) + "\"")); err != nil {
		t.Logf("invalid yourColor")
		t.FailNow()
	}

	return gameStateMessage{
		Type:      typ,
		Board:     *b,
		Version:   version,
		Result:    *r,
		Plies:     plies,
		ToPlay:    toPlay,
		YourColor: yourColor,
	}
}

func tryMachConnected(t *testing.T, m map[string]any) machConnectedMessage {
	typ := tryType(t, "mach/connected", tryGet(t, m, "type").(string))
	id := tryId(t, tryGet(t, m, "id").(string))
	color := tryColor(t, tryGet(t, m, "yourColor").(string))

	return machConnectedMessage{
		Type:      typ,
		Id:        id,
		YourColor: color,
	}
}

// TODO: also test situations where the server should return an error

func TestMachGame(t *testing.T) {
	cli, conn := getClientAndConn(t)

	go cli.handleFirstMessage()

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "mach/new",
		"data": map[string]any{
			"humanColor":  "white",
			"heuristic":   "WeightedCount",
			"timeLimitMs": 100,
		},
	}))

	tryMachConnected(t, tryRead(t, conn))

	for {
		m := tryRead(t, conn)
		if m["type"] == "error" {
			t.Logf("received error: %v", m["message"])
			t.FailNow()
		}
		s := tryState(t, m)
		if s.Result.Over() {
			break
		}
		if s.ToPlay != core.WhiteColor {
			continue
		}
		r := rand.Intn(len(s.Plies))
		json := tryJson(t, map[string]any{
			"type": "ply",
			"data": map[string]any{
				"version": s.Version,
				"index":   r,
			},
		})
		trySend(t, conn, json)
	}

	assertClosed(t, cli)
}

func TestMachGameConnect(t *testing.T) {
	var connected machConnectedMessage

	{
		cli, conn := getClientAndConn(t)

		go cli.handleFirstMessage()

		trySend(t, conn, tryJson(t, map[string]any{
			"type": "mach/new",
			"data": map[string]any{
				"humanColor":  "white",
				"heuristic":   "WeightedCount",
				"timeLimitMs": 100,
			},
		}))

		connected = tryMachConnected(t, tryRead(t, conn))
	}

	{
		cli, conn := getClientAndConn(t)

		go cli.handleFirstMessage()

		trySend(t, conn, tryJson(t, map[string]any{
			"type": "mach/connect",
			"data": map[string]any{
				"id": connected.Id,
			},
		}))

		tryMachConnected(t, tryRead(t, conn))
		tryState(t, tryRead(t, conn))

		conn.Close()
		assertClosed(t, cli)
	}

}

func tryHumanCreated(t *testing.T, m map[string]any) humanCreatedMessage {
	typ := tryType(t, "human/created", tryGet(t, m, "type").(string))
	id := tryId(t, tryGet(t, m, "id").(string))
	yourColor := tryColor(t, tryGet(t, m, "yourColor").(string))

	yourToken := tryGet(t, m, "yourToken").(string)
	opponentToken := tryGet(t, m, "opponentToken").(string)

	return humanCreatedMessage{
		Type:          typ,
		Id:            id,
		YourColor:     yourColor,
		YourToken:     yourToken,
		OpponentToken: opponentToken,
	}
}

func tryHumanConnected(t *testing.T, m map[string]any) humanConnectedMessage {
	typ := tryType(t, "human/connected", tryGet(t, m, "type").(string))
	id := tryId(t, tryGet(t, m, "id").(string))
	yourColor := tryColor(t, tryGet(t, m, "yourColor").(string))
	yourToken := tryGet(t, m, "yourToken").(string)
	return humanConnectedMessage{
		Type:      typ,
		Id:        id,
		YourColor: yourColor,
		YourToken: yourToken,
	}
}

func tryError(t *testing.T, m map[string]any) stringMessage {
	typ := tryType(t, "error", tryGet(t, m, "type").(string))
	message := tryGet(t, m, "message").(string)
	return stringMessage{
		Type:    typ,
		Message: message,
	}
}

func TestHumanGame(t *testing.T) {
	wcli, wconn := getClientAndConn(t)
	go wcli.handleFirstMessage()

	trySend(t, wconn, tryJson(t, map[string]any{
		"type": "human/new",
		"data": map[string]any{
			"color": "white",
		},
	}))

	created := tryHumanCreated(t, tryRead(t, wconn))

	bcli, bconn := getClientAndConn(t)
	go bcli.handleFirstMessage()

	trySend(t, bconn, tryJson(t, map[string]any{
		"type": "human/connect",
		"data": map[string]any{
			"id":    created.Id,
			"token": created.OpponentToken,
		},
	}))

	connected := tryHumanConnected(t, tryRead(t, bconn))

	if connected.YourColor != created.YourColor.Opposite() {
		t.Log("not the opponent color")
		t.FailNow()
	}

	for {
		// TODO check if they got the same states
		state := tryState(t, tryRead(t, wconn))
		tryState(t, tryRead(t, bconn))

		if state.Result.Over() {
			break
		}

		connToPlay := wconn
		if state.ToPlay == core.BlackColor {
			connToPlay = bconn
		}

		r := rand.Intn(len(state.Plies))
		trySend(t, connToPlay, tryJson(t, map[string]any{
			"type": "ply",
			"data": map[string]any{
				"version": state.Version,
				"index":   r,
			},
		}))
	}

	assertClosed(t, wcli)
	assertClosed(t, bcli)
}

func TestClientError(t *testing.T) {
	incoming := make(chan []byte)
	outgoing := make(chan []byte)
	c := &client{incoming, outgoing}
	content := "test"
	go c.error(content)
	bs, ok := <-outgoing
	if !ok {
		t.Logf("channel prematurely closed")
		t.FailNow()
	}
	var msg stringMessage
	if err := json.Unmarshal(bs, &msg); err != nil {
		t.Logf("failed to unmarshal")
		t.FailNow()
	}
	if msg.Type != "error" {
		t.Logf("invalid type")
		t.FailNow()
	}
	if msg.Message != content {
		t.Logf("invalid content")
		t.FailNow()
	}
}

func TestUnknownType(t *testing.T) {
	incoming := make(chan []byte)
	outgoing := make(chan []byte)
	c := &client{incoming, outgoing}

	go c.handleFirstMessage()

	weirdMessage := map[string]any{"type": "what"}
	weirdBs, err := json.Marshal(weirdMessage)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	incoming <- weirdBs

	var response stringMessage
	responseBs := <-outgoing
	if err := json.Unmarshal(responseBs, &response); err != nil {
		t.Log(err)
		t.FailNow()
	}

	if response.Type != "error" {
		t.Log("invalid type, wanted error")
		t.FailNow()
	}
}

func TestMachInvalidId(t *testing.T) {
	var cli *client
	var conn *websocket.Conn
	cli, conn = getClientAndConn(t)

	go cli.handleFirstMessage()

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "mach/new",
		"data": map[string]any{
			"humanColor":  "white",
			"heuristic":   "WeightedCount",
			"timeLimitMs": 100,
		},
	}))

	// ignore
	tryRead(t, conn)

	// disconnect
	conn.Close()

	cli, conn = getClientAndConn(t)

	go cli.handleFirstMessage()

	randId, err := uuid.NewRandom()
	if err != nil {
		t.Log("failed to generated random uuid", err)
		t.FailNow()
	}

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "mach/connect",
		"data": map[string]any{
			"id": randId,
		},
	}))

	response := tryError(t, tryRead(t, conn))

	if !strings.Contains(response.Message, "machine game not found") {
		t.Logf("expected 'not found' error response, got %q", response.Message)
		t.FailNow()
	}
}

func TestHumanInvalidIdAndToken(t *testing.T) {
	var cli *client
	var conn *websocket.Conn

	cli, conn = getClientAndConn(t)

	go cli.handleFirstMessage()

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "human/new",
		"data": map[string]any{
			"color": "white",
		},
	}))

	created := tryHumanCreated(t, tryRead(t, conn))

	// disconnect
	conn.Close()

	cli, conn = getClientAndConn(t)
	go cli.handleFirstMessage()

	// wrong id

	randId, err := uuid.NewRandom()
	if err != nil {
		t.Log("failed to generated random uuid", err)
		t.FailNow()
	}

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "human/connect",
		"data": map[string]any{
			"id": randId,
		},
	}))

	response := tryError(t, tryRead(t, conn))

	if !strings.Contains(response.Message, "human game not found") {
		t.Logf("expected 'not found' error response, got %q", response.Message)
		t.FailNow()
	}

	// right id, but invalid token

	randToken, err := genToken()
	if err != nil {
		t.Logf("failed to generate token: %s", err)
		t.FailNow()
	}

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "human/connect",
		"data": map[string]any{
			"id":    created.Id,
			"token": randToken,
		},
	}))

	response = tryError(t, tryRead(t, conn))

	if !strings.Contains(response.Message, "invalid token") {
		t.Logf("expected 'invalid token' error response, got %q", response.Message)
		t.FailNow()
	}
}

func TestHumanCreateAndReconnect(t *testing.T) {
	var cli *client
	var conn *websocket.Conn

	cli, conn = getClientAndConn(t)
	go cli.handleFirstMessage()

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "human/new",
		"data": map[string]any{
			"color": "white",
		},
	}))

	created := tryHumanCreated(t, tryRead(t, conn))

	// disconnect
	conn.Close()

	cli, conn = getClientAndConn(t)
	go cli.handleFirstMessage()

	trySend(t, conn, tryJson(t, map[string]any{
		"type": "human/connect",
		"data": map[string]any{
			"id":    created.Id,
			"token": created.YourToken,
		},
	}))

	tryHumanConnected(t, tryRead(t, conn))
}
