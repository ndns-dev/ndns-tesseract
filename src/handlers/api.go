package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/ndns-dev/ndns-tesseract/src/services"
	customTypes "github.com/ndns-dev/ndns-tesseract/src/types"
	"github.com/ndns-dev/ndns-tesseract/src/utils"
)

// HandleAPIGatewayEvent는 API Gateway로부터의 요청을 처리합니다.
func HandleAPIGatewayEvent(ctx context.Context, e events.APIGatewayProxyRequest) (interface{}, error) {
	var queueState customTypes.OcrQueueState
	contentType := strings.ToLower(e.Headers["Content-Type"])

	log.Printf("Received API Gateway event - Content-Type: %s", contentType)

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		formData := utils.ParseFormURLEncoded(e.Body)
		log.Printf("Parsed form data: %+v", formData)

		// queueState에 값 할당
		queueState.ReqId = formData["reqId"]
		queueState.JobId = formData["jobId"]
		queueState.CurrentPosition = customTypes.OcrPosition(formData["currentPosition"])
		queueState.Is2025OrLater, _ = strconv.ParseBool(formData["is2025OrLater"])
		queueState.RequestedAt, _ = time.Parse(time.RFC3339, formData["requestedAt"])
		queueState.CrawlResult = &customTypes.CrawlResult{
			Url:              formData["crawlResult.url"],
			FirstParagraph:   formData["crawlResult.firstParagraph"],
			LastParagraph:    formData["crawlResult.lastParagraph"],
			Content:          formData["crawlResult.content"],
			FirstImageUrl:    formData["crawlResult.firstImageUrl"],
			LastImageUrl:     formData["crawlResult.lastImageUrl"],
			FirstStickerUrl:  formData["crawlResult.firstStickerUrl"],
			SecondStickerUrl: formData["crawlResult.secondStickerUrl"],
			LastStickerUrl:   formData["crawlResult.lastStickerUrl"],
		}

		// 디버그 로깅
		log.Printf("Constructed queueState from form data - ReqId: %s, JobId: %s, Position: %s, URL: %s",
			queueState.ReqId,
			queueState.JobId,
			queueState.CurrentPosition,
			queueState.CrawlResult.Url)

	} else {
		log.Printf("Attempting to parse request body as JSON")
		err := json.Unmarshal([]byte(e.Body), &queueState)
		if err != nil {
			return utils.Response(utils.ErrorHandler(ctx, fmt.Errorf("could not unmarshal HTTP request body: %w", err), queueState.JobId, queueState.CrawlResult.Url, "HTTPRequestUnmarshal"))
		}
		log.Printf("Successfully parsed JSON request body")
	}

	errResp, err := services.HandleOcrWorkflow(ctx, queueState)
	return utils.Response(errResp, err)
}
