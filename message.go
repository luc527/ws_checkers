package main

import (
	"encoding/json"

	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

func parseHeuristic(h string) (minimax.Heuristic, bool) {
	if h == "unweightedCount" {
		return minimax.UnweightedCountHeuristic, true
	}
	if h == "weightedCount" {
		return minimax.WeightedCountHeuristic, true
	}
	return nil, false
}

type messageEnvelope struct {
	T   string           `json:"type"`
	Raw *json.RawMessage `json:"data"`
}

type gameStateMessageData struct {
	GameId string     `json:"gameId"`
	Board  string     `json:"board"`
	Result string     `json:"result"`
	ToPlay string     `json:"toPlay"`
	Plies  []core.Ply `json:"plies"`
}
