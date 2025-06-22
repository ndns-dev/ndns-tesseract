package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

var webhookURL string

func init() {
	webhookURL = os.Getenv("ERROR_WEBHOOK_URL")
	if webhookURL == "" {
		log.Println("WARNING: ERROR_WEBHOOK_URL environment variable is not set. Error notifications will not be sent via webhook.")
	}
}

type WebhookPayload struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	JobID     string `json:"jobId,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
	Source    string `json:"source"`
}

// NotifyErrorToWebhook: 웹훅으로 에러 메시지를 비동기적으로 전송합니다.
// 이 함수는 'main.go'의 핸들러에서만 호출될 것을 상정합니다.
func NotifyErrorToWebhook(ctx context.Context, level string, errMsg string, jobID string, imageURL string, source string) {
	if webhookURL == "" {
		return
	}

	payload := WebhookPayload{
		Level:     level,
		Message:   errMsg,
		Timestamp: time.Now().Format(time.RFC3339),
		JobID:     jobID,
		ImageURL:  imageURL,
		Source:    source,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR (Webhook GoRoutine): Failed to marshal webhook payload: %v", err)
		return
	}

	go func() {
		localCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(localCtx, "POST", webhookURL, bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Printf("ERROR (Webhook GoRoutine): Failed to create webhook request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("ERROR (Webhook GoRoutine): Failed to send webhook request: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("ERROR (Webhook GoRoutine): Webhook returned non-success status code: %d, Response: %s", resp.StatusCode, resp.Status)
		} else {
			log.Printf("INFO: Webhook notification sent successfully.")
		}
	}()
}
