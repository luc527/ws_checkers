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

// Outgoing
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

// Outgoing
type gameStateMessage struct {
	Type        string          `json:"type"`
	Version     int             `json:"version"`
	Board       core.Board      `json:"board"`
	WhiteToPlay bool            `json:"whiteToPlay"`
	Result      core.GameResult `json:"result"`
	Plies       []core.Ply      `json:"plies"`
}

// Incoming
type plyMessage struct {
	Version  int `json:"version"`
	PlyIndex int `json:"ply"`
}

// Incoming
type newMachineGameMessage struct {
	HumanColor        core.Color       `json:"humanColor"`
	CapturesMandatory core.CaptureRule `json:"captureMandatory"`
	BestMandatory     core.BestRule    `json:"bestMandatory"`
	TimeLimitMs       int              `json:"timeLimitMs,omitempty"`
	Heuristic         string           `json:"heuristic,omitempty"`
	// TODO: turn heuristic into minimax.Heuristic, implement MarshalJSON and UnmarshalJSON
}

func gameStateMessageFrom(s gameState) gameStateMessage {
	return gameStateMessage{
		Type:        "state",
		Version:     s.v,
		Board:       s.board,
		WhiteToPlay: s.toPlay == core.WhiteColor,
		Result:      s.result,
		Plies:       s.plies,
	}
}

// Outgoing
type gameIdMessage struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

func gameIdMessageFrom(id uuid.UUID) gameIdMessage {
	return gameIdMessage{
		Type: "id",
		Id:   id.String(),
	}
}

// Incoming
type reconnectMachineGameMessage struct {
	Id string `json:"id"`
}
