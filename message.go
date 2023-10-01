package main

import "encoding/json"

type messageEnvelope struct {
	T   string           `json:"type"`
	Raw *json.RawMessage `json:"data"`
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
