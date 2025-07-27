package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/ndns-dev/ndns-tesseract/src/services"
	customTypes "github.com/ndns-dev/ndns-tesseract/src/types"
	"github.com/ndns-dev/ndns-tesseract/src/utils"
)

// getString은 map에서 안전하게 문자열 값을 추출합니다.
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// HandleSQSEvent는 SQS로부터의 메시지를 처리합니다.
func HandleSQSEvent(ctx context.Context, e events.SQSEvent) (interface{}, error) {
	for _, record := range e.Records {
		var queueState customTypes.OcrQueueState
		var bodyMap map[string]interface{}

		err := json.Unmarshal([]byte(record.Body), &bodyMap)
		if err != nil {
			return utils.Response(utils.ErrorHandler(ctx, fmt.Errorf("could not unmarshal SQS message body: %w", err), "", "", "SQSMessageUnmarshal"))
		}

		// queueState에 값 할당
		queueState.ReqId = getString(bodyMap, "reqId")
		queueState.JobId = getString(bodyMap, "jobId")
		queueState.CurrentPosition = customTypes.OcrPosition(getString(bodyMap, "currentPosition"))
		queueState.Is2025OrLater, _ = strconv.ParseBool(getString(bodyMap, "is2025OrLater"))
		requestedAt := getString(bodyMap, "requestedAt")
		if requestedAt != "" {
			queueState.RequestedAt, _ = time.Parse(time.RFC3339, requestedAt)
		}

		// CrawlResult 파싱
		if crawlResultMap, ok := bodyMap["crawlResult"].(map[string]interface{}); ok {
			queueState.CrawlResult = &customTypes.CrawlResult{
				Url:              getString(crawlResultMap, "url"),
				FirstParagraph:   getString(crawlResultMap, "firstParagraph"),
				LastParagraph:    getString(crawlResultMap, "lastParagraph"),
				Content:          getString(crawlResultMap, "content"),
				FirstImageUrl:    getString(crawlResultMap, "firstImageUrl"),
				LastImageUrl:     getString(crawlResultMap, "lastImageUrl"),
				FirstStickerUrl:  getString(crawlResultMap, "firstStickerUrl"),
				SecondStickerUrl: getString(crawlResultMap, "secondStickerUrl"),
				LastStickerUrl:   getString(crawlResultMap, "lastStickerUrl"),
			}
		}

		// 디버그 로깅
		log.Printf("Parsed queueState from SQS message - ReqId: %s, JobId: %s, Position: %s, URL: %s",
			queueState.ReqId,
			queueState.JobId,
			queueState.CurrentPosition,
			queueState.CrawlResult.Url)
		utils.WebhookLog("ndns-tesseract: SQS RECEIVED: %s", queueState.JobId)
		result, err := services.HandleOcrWorkflow(ctx, queueState)
		if err != nil {
			log.Printf("Error processing record: %v", err)
			continue
		}
		log.Printf("Successfully processed record: %s", result.JobId)
	}

	return utils.Response(nil, nil)
}
