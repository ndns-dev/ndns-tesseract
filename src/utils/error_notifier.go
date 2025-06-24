package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var webhookURL string
var httpClient *http.Client

func init() {
	webhookURL = os.Getenv("ERROR_WEBHOOK_URL")
	if webhookURL == "" {
		log.Println("WARNING: ERROR_WEBHOOK_URL environment variable is not set. Error notifications will not be sent via webhook.")
	}

	// 커스텀 Transport 설정
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,  // 연결 타임아웃
			KeepAlive: 30 * time.Second, // Keep-Alive 시간
		}).DialContext,
		MaxIdleConns:          100,              // 최대 유휴 연결 수
		IdleConnTimeout:       90 * time.Second, // 유휴 연결 타임아웃
		TLSHandshakeTimeout:   5 * time.Second,  // TLS 핸드쉐이크 타임아웃
		ExpectContinueTimeout: 1 * time.Second,  // Expect: 100-continue에 대한 타임아웃
		ResponseHeaderTimeout: 5 * time.Second,  // 응답 헤더 타임아웃
	}

	// HTTP 클라이언트 초기화
	httpClient = &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second, // 전체 요청 타임아웃
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
		log.Printf("ERROR (Webhook): Failed to marshal webhook payload: %v", err)
		return
	}

	go func() {
		// 컨텍스트 타임아웃 설정 (10초)
		localCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(localCtx, "POST", webhookURL, bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Printf("ERROR (Webhook): Failed to create webhook request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("ERROR (Webhook): Failed to send webhook request: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("ERROR (Webhook): Discord returned non-success status code: %d", resp.StatusCode)
		} else {
			log.Printf("INFO (Webhook): Notification sent successfully")
		}
	}()
}
