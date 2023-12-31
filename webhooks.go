package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

type webhookRequestBody struct {
	Mode      string          `json:"mode"`
	Id        uuid.UUID       `json:"id"`
	Result    core.GameResult `json:"result"`
	Timestamp int64           `json:"timestamp"`
}

func notifyWebhooks(mode gameMode, id uuid.UUID, state gameState, urls []string) {
	body := webhookRequestBody{
		Mode:      mode.String(),
		Id:        id,
		Result:    state.result,
		Timestamp: time.Now().UnixMilli(),
	}
	bytes, err := json.Marshal(body)
	if err != nil {
		log.Printf("failed to marshal webhook request body: %v", err)
		return
	}
	for _, url := range urls {
		log.Printf("notifying webhook %v of %v game %v result %v", url, mode, id, state)
		go webhookSend(url, bytes)
	}
}

func webhookSend(url string, data []byte) {
	reader := bytes.NewReader(data)
	resp, err := http.Post(url, "application/json", reader)
	if err != nil {
		// TODO retry with exponential backoff + jitter
		log.Printf("webhook send failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		log.Printf("webhook %v ok", url)
	} else {
		// TODO retry with exponential backoff + jitter
		log.Printf("webhook %v failed", url)
	}
}
