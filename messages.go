package main

import "github.com/luc527/go_checkers/core"

// TODO implementar MarshalJSON para core.Ply
// o que vai requerir MarshalJSON para core.Instruction
// fazer no pacote 'core' mesmo
// no final dá pra descomentar o Plies

type gameStateMessage struct {
	Type        string `json:"type"`
	Version     int    `json:"version"`
	Board       string `json:"board"`
	WhiteToPlay bool   `json:"whiteToPlay"`
	Result      string `json:"result"`
	// Plies       []core.Ply `json:"plies"`
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
	}
}
