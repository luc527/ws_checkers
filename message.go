package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

type against byte

const (
	againstHuman = against(iota)
	againstAI
)

func (a against) ai() bool {
	return a == againstAI
}

func (a against) human() bool {
	return a == againstHuman
}

func parseAgainst(a string) (against, bool) {
	if a == "human" {
		return againstHuman, true
	}
	if a == "ai" {
		return againstAI, true
	}
	return 0, false
}

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

type newGameMessageData struct {
	Against           string `json:"against"`
	CapturesMandatory bool   `json:"capturesMandatory"`
	BestMandatory     bool   `json:"bestMandatory"`
	AITimeLimitMs     int    `json:"aiTimeLimitMs,omitempty"`
	AIHeuristic       string `json:"aiHeuristic,omitempty"`
}

type newGameMessage struct {
	against
	captureRule core.CaptureRule
	bestRule    core.BestRule
	aiTimeLimit time.Duration
	aiHeuristic minimax.Heuristic
}

type stringMessage struct {
	T    string `json:"type"`
	Text string `json:"message"`
}

func errorMessage(err string) stringMessage {
	return stringMessage{
		T:    "error",
		Text: err,
	}
}

func parseNewGameMessage(msg messageEnvelope) (*newGameMessage, error) {
	data := newGameMessageData{}
	if msg.Raw == nil {
		return nil, fmt.Errorf("new game message: no data")
	}
	if err := json.Unmarshal(*msg.Raw, &data); err != nil {
		return nil, err
	}

	against, ok := parseAgainst(data.Against)
	if !ok {
		return nil, fmt.Errorf("new game message: invalid 'against' value %q", data.Against)
	}

	captureRule := core.CaptureRule(data.CapturesMandatory)
	bestRule := core.BestRule(data.BestMandatory)

	var aiTimeLimit time.Duration
	var aiHeuristic minimax.Heuristic

	if against.ai() {
		aiTimeLimit = 3 * time.Second
		aiHeuristic = minimax.WeightedCountHeuristic

		if data.AITimeLimitMs > 0 {
			aiTimeLimit = time.Duration(data.AITimeLimitMs * int(time.Millisecond))
		}

		if data.AIHeuristic != "" {
			aiHeuristic, ok = parseHeuristic(data.AIHeuristic)
			if !ok {
				return nil, fmt.Errorf("new game message: invalid 'aiHeuristic' value %q", data.AIHeuristic)
			}
		}
	}

	newGameMsg := newGameMessage{
		against:     against,
		captureRule: captureRule,
		bestRule:    bestRule,
		aiTimeLimit: aiTimeLimit,
		aiHeuristic: aiHeuristic,
	}
	return &newGameMsg, nil
}
