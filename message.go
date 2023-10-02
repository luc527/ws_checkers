package main

import (
	"encoding/json"
	"fmt"
	"time"

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

type newAIGameMessageData struct {
	CapturesMandatory bool   `json:"capturesMandatory"`
	BestMandatory     bool   `json:"bestMandatory"`
	TimeLimitMs       int    `json:"timeLimitMs,omitempty"`
	Heuristic         string `json:"heuristic,omitempty"`
}

type newMachGameMessage struct {
	captureRule core.CaptureRule
	bestRule    core.BestRule
	timeLimit   time.Duration
	heuristic   minimax.Heuristic
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

func parseNewMachGameMessage(msg messageEnvelope) (*newMachGameMessage, error) {
	data := newAIGameMessageData{}
	if msg.Raw == nil {
		return nil, fmt.Errorf("new game message: no data")
	}
	if err := json.Unmarshal(*msg.Raw, &data); err != nil {
		return nil, err
	}

	captureRule := core.CaptureRule(data.CapturesMandatory)
	bestRule := core.BestRule(data.BestMandatory)

	var timeLimit time.Duration
	var heuristic minimax.Heuristic

	timeLimit = 3 * time.Second
	heuristic = minimax.WeightedCountHeuristic

	if data.TimeLimitMs > 0 {
		timeLimit = time.Duration(data.TimeLimitMs * int(time.Millisecond))
	}

	if data.Heuristic != "" {
		var ok bool
		heuristic, ok = parseHeuristic(data.Heuristic)
		if !ok {
			return nil, fmt.Errorf("new game message: invalid 'Heuristic' value %q", data.Heuristic)
		}
	}

	newGameMsg := newMachGameMessage{
		captureRule: captureRule,
		bestRule:    bestRule,
		timeLimit:   timeLimit,
		heuristic:   heuristic,
	}
	return &newGameMsg, nil
}
