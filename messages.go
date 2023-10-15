package main

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

// Envelope for all incoming messages
type messageEnvelope struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"data"`
}

type stringMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type idMessage struct {
	Type  string    `json:"type"`
	Id    uuid.UUID `json:"id"`
	Token string    `json:"token,omitempty"`
}

func errorMessage(err string) stringMessage {
	return stringMessage{
		Type:    "error",
		Message: err,
	}
}

func machIdMessage(id uuid.UUID) idMessage {
	return idMessage{
		Type: "id",
		Id:   id,
	}
}

type machNewData struct {
	CapturesMandatory core.CaptureRule `json:"captureRule"`
	BestMandatory     core.BestRule    `json:"bestRule"`
	HumanColor        core.Color       `json:"humanColor"`
	Heuristic         string           `json:"heuristic"`
	TimeLimitMs       int              `json:"timeLimitMs"`
}

type gameStateMessage struct {
	Type      string          `json:"type"`
	Board     core.Board      `json:"board"`
	Version   int             `json:"version"`
	Result    core.GameResult `json:"result"`
	ToPlay    core.Color      `json:"toPlay"`
	Plies     []core.Ply      `json:"plies"`
	YourColor core.Color      `json:"yourColor"`
}

func gameStateMessageFrom(s gameState, player core.Color) gameStateMessage {
	return gameStateMessage{
		Type:      "state",
		Board:     s.board,
		Version:   s.v,
		Result:    s.result,
		ToPlay:    s.toPlay,
		Plies:     s.plies,
		YourColor: player,
	}
}

type plyData struct {
	Version int `json:"version"`
	Index   int `json:"ply"`
}

type machConnectData struct {
	Id uuid.UUID `json:"id"`
}
