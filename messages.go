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

type machConnectedMessage struct {
	Type      string     `json:"type"`
	Id        uuid.UUID  `json:"id"`
	YourColor core.Color `json:"yourColor"`
}

type humanCreatedMessage struct {
	Type          string     `json:"type"`
	Id            uuid.UUID  `json:"id"`
	YourColor     core.Color `json:"yourColor"`
	YourToken     string     `json:"yourToken"`
	OpponentToken string     `json:"opponentToken"`
}

type humanConnectedMessage struct {
	Type      string     `json:"type"`
	Id        uuid.UUID  `json:"id"`
	YourColor core.Color `json:"yourColor"`
	YourToken string     `json:"yourToken"`
}

func errorMessage(err string) stringMessage {
	return stringMessage{
		Type:    "error",
		Message: err,
	}
}

func machConnectedMessageFrom(color core.Color, id uuid.UUID) machConnectedMessage {
	return machConnectedMessage{
		Type:      "mach/connected",
		Id:        id,
		YourColor: color,
	}
}

func humanCreatedMessageFrom(color core.Color, id uuid.UUID, yourToken string, OpponentToken string) humanCreatedMessage {
	return humanCreatedMessage{
		Type:          "human/created",
		Id:            id,
		YourColor:     color,
		YourToken:     yourToken,
		OpponentToken: OpponentToken,
	}
}

func humanConnectedMessageFrom(color core.Color, id uuid.UUID, token string) humanConnectedMessage {
	return humanConnectedMessage{
		Type:      "human/connected",
		Id:        id,
		YourColor: color,
		YourToken: token,
	}
}

type machNewData struct {
	HumanColor  core.Color `json:"humanColor"`
	Heuristic   string     `json:"heuristic"`
	TimeLimitMs int        `json:"timeLimitMs"`
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
		Version:   s.version,
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

type humanNewData struct {
	Color core.Color `json:"color"`
}

type humanConnectData struct {
	Id    uuid.UUID `json:"id"`
	Token string    `json:"token"`
}
