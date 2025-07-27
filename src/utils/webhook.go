package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type WebhookPayload struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	JobId     string `json:"JobId,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
	Source    string `json:"source"`
}

func formatMessage(format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)

	// 구조체나 맵이 포함된 경우 JSON으로 예쁘게 포맷팅
	for _, arg := range args {
		if _, ok := arg.(map[string]interface{}); ok {
			if jsonBytes, err := json.MarshalIndent(arg, "", "  "); err == nil {
				message = string(jsonBytes)
			}
		} else {
			// 구조체인 경우
			if jsonBytes, err := json.MarshalIndent(arg, "", "  "); err == nil {
				message = string(jsonBytes)
			}
		}
	}

	return message
}

func WebhookLog(format string, args ...interface{}) {
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	message := formatMessage(format, args...)

	// Discord 웹훅 메시지 구조
	payload := map[string]interface{}{
		"content": "```json\n" + message + "\n```",
		"embeds": []map[string]interface{}{
			{
				"title":     format,
				"color":     0x00ff00,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("웹훅 JSON 생성 실패: %v", err)
		return
	}

	// HTTP POST 요청
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("웹훅 전송 실패: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("웹훅 전송 실패 (상태 코드: %d): %s", resp.StatusCode, string(body))
	}
}
