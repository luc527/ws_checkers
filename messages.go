package main

import (
	"encoding/json"

	"github.com/luc527/go_checkers/core"
)

type messageEnvelope struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"data"`
}

type gameStateMessage struct {
	Type        string     `json:"type"`
	Version     int        `json:"version"`
	Board       string     `json:"board"`
	WhiteToPlay bool       `json:"whiteToPlay"`
	Result      string     `json:"result"`
	Plies       []core.Ply `json:"plies"`
}

type plyMessage struct {
	Version  int `json:"version"`
	PlyIndex int `json:"ply"`
}

func gameStateMessageFrom(s gameState) gameStateMessage {
	return gameStateMessage{
		Type:        "state",
		Version:     s.v,
		Board:       s.board.Serialize(),
		WhiteToPlay: s.toPlay == core.WhiteColor,
		Result:      s.result.String(),
		Plies:       s.plies,
	}
}
