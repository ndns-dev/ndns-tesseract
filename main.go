package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	ocr "github.com/ndns-dev/ndns-tesseract/src/ocr"
	customTypes "github.com/ndns-dev/ndns-tesseract/src/types"
	"github.com/ndns-dev/ndns-tesseract/src/utils"
)

func handleRequest(ctx context.Context, event interface{}) (interface{}, error) {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return utils.Response(utils.ErrorHandler(ctx, fmt.Errorf("failed to marshal event: %w", err), "", "", "EventMarshal"))
	}

	var eventMap map[string]interface{}
	if err := json.Unmarshal(eventBytes, &eventMap); err == nil {
		// API Gateway 이벤트인지 확인
		if _, hasBody := eventMap["body"]; hasBody {
			if _, hasHeaders := eventMap["headers"]; hasHeaders {
				// API Gateway 이벤트로 처리
				var apiEvent events.APIGatewayProxyRequest
				if err := json.Unmarshal(eventBytes, &apiEvent); err == nil {
					return handleAPIGatewayEvent(ctx, apiEvent)
				}
			}
		}
	}

	// SQS 이벤트 처리
	if sqsEvent, ok := event.(events.SQSEvent); ok {
		return handleSQSEvent(ctx, sqsEvent)
	}

	return utils.Response(utils.ErrorHandler(ctx, fmt.Errorf("unsupported event type"), "", "", "EventTypeHandler"))
}

func handleAPIGatewayEvent(ctx context.Context, e events.APIGatewayProxyRequest) (interface{}, error) {
	var queueState customTypes.OcrQueueState
	contentType := strings.ToLower(e.Headers["Content-Type"])

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		formData := utils.ParseFormURLEncoded(e.Body)
		// queueState에 값 할당
		queueState.JobId = formData["JobId"]
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
		log.Printf("Parsed form data - JobId: %s, URL: %s, Position: %s",
			queueState.JobId,
			queueState.CrawlResult.Url,
			queueState.CurrentPosition)

	} else {
		err := json.Unmarshal([]byte(e.Body), &queueState)
		if err != nil {
			return utils.Response(utils.ErrorHandler(ctx, fmt.Errorf("could not unmarshal HTTP request body: %w", err), queueState.JobId, queueState.CrawlResult.Url, "HTTPRequestUnmarshal"))
		}
	}

	errResp, err := processOCRRequest(ctx, queueState)
	return utils.Response(errResp, err)
}

func handleSQSEvent(ctx context.Context, e events.SQSEvent) (interface{}, error) {
	if len(e.Records) == 0 {
		return utils.ErrorHandler(ctx, fmt.Errorf("no records in SQS event"), "", "", "SQSEventHandler")
	}

	record := e.Records[0]
	var queueState customTypes.OcrQueueState

	// SQS 메시지 바디를 파싱하여 map으로 변환
	var bodyMap map[string]interface{}
	if err := json.Unmarshal([]byte(record.Body), &bodyMap); err != nil {
		return utils.ErrorHandler(ctx, fmt.Errorf("could not unmarshal SQS message body: %w", err), "", "", "SQSMessageUnmarshal")
	}

	// queueState에 값 할당
	if jobId, ok := bodyMap["JobId"].(string); ok {
		queueState.JobId = jobId
	}
	if currentPosition, ok := bodyMap["currentPosition"].(string); ok {
		queueState.CurrentPosition = customTypes.OcrPosition(currentPosition)
	}
	if is2025OrLater, ok := bodyMap["is2025OrLater"].(bool); ok {
		queueState.Is2025OrLater = is2025OrLater
	}
	if requestedAt, ok := bodyMap["requestedAt"].(string); ok {
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
	log.Printf("Parsed SQS message - JobId: %s, URL: %s, Position: %s",
		queueState.JobId,
		queueState.CrawlResult.Url,
		queueState.CurrentPosition)

	return processOCRRequest(ctx, queueState)
}

// getString은 map에서 안전하게 문자열 값을 추출하는 헬퍼 함수입니다.
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func processOCRRequest(ctx context.Context, queueState customTypes.OcrQueueState) (*customTypes.ErrorResponse, error) {
	if queueState.JobId == "" {
		return utils.ErrorHandler(ctx, fmt.Errorf("missing JobId"), "", "", "RequestValidation")
	}

	// CurrentPosition에 해당하는 이미지 URL 가져오기
	imageUrl := queueState.CrawlResult.GetImageUrlByPosition(queueState.CurrentPosition)
	if imageUrl == "" {
		return utils.ErrorHandler(ctx, fmt.Errorf("no image URL found for position: %s", queueState.CurrentPosition), queueState.JobId, queueState.CrawlResult.Url, "RequestValidation")
	}

	log.Printf("Processing JobId: %s, URL: %s, Image URL: %s", queueState.JobId, queueState.CrawlResult.Url, imageUrl)

	// OCR 수행을 위한 요청 바디 구성
	ocrRequestPayload := customTypes.OCRRequest{
		ImageUrl: imageUrl,
	}

	// OCRRequestPayload를 JSON 바이트 배열로 마샬링
	requestBodyBytes, err := json.Marshal(ocrRequestPayload)
	if err != nil {
		return utils.ErrorHandler(ctx, fmt.Errorf("failed to marshal OCR request payload: %w", err), queueState.JobId, queueState.CrawlResult.Url, "MarshalOCRRequest")
	}

	// OCR 수행
	ocrResultDetails, ocrErr := ocr.PerformOCR(requestBodyBytes)

	// 최종 결과를 저장할 customTypes.OCRJobDetails 객체 생성
	finalResult := customTypes.OCRJobDetails{
		JobId:     queueState.JobId,
		ImageURL:  imageUrl,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if ocrErr != nil {
		finalResult.Status = customTypes.JobStatusFailed
		finalResult.OCRText = ""
		finalResult.Error = ocrErr.Error()
	} else {
		finalResult.Status = ocrResultDetails.Status
		finalResult.OCRText = ocrResultDetails.OCRText
		finalResult.Error = ocrResultDetails.Error
	}

	// OCR JobDetails 테이블 업데이트
	if err := utils.UpdateOCRJobDetails(ctx, finalResult); err != nil {
		return utils.ErrorHandler(ctx, fmt.Errorf("failed to update OCRJobDetails for JobId %s: %w", queueState.JobId, err), queueState.JobId, queueState.CrawlResult.Url, "UpdateOCRJobDetails")
	}
	log.Printf("OCR JobDetails for JobId %s updated to status: %s", queueState.JobId, finalResult.Status)

	// OCR 성공 시에만 캐싱 테이블에 저장
	if finalResult.Status == customTypes.JobStatusCompleted {
		ocrResult := customTypes.OCRResults{
			ImageURL:  imageUrl,
			OCRText:   finalResult.OCRText,
			Status:    customTypes.OCRStatus(customTypes.JobStatusCompleted),
			Error:     finalResult.Error,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if err := utils.SaveOCRResult(ctx, ocrResult); err != nil {
			log.Printf("WARN: Failed to save OCR result to cache table for URL %s (JobId: %s): %v", queueState.CrawlResult.Url, queueState.JobId, err)
		} else {
			log.Printf("OCR result for URL %s (JobId: %s) saved to cache table.", queueState.CrawlResult.Url, queueState.JobId)
		}
	}

	// OCR이 실패했을 경우 에러 반환
	if finalResult.Status == customTypes.JobStatusFailed {
		return &customTypes.ErrorResponse{
			Message:   finalResult.Error,
			JobId:     finalResult.JobId,
			ImageURL:  imageUrl,
			ErrorCode: "OCRExecutionFailed",
		}, fmt.Errorf(finalResult.Error)
	}

	// 성공 시 nil 반환
	return nil, nil
}

func main() {
	lambda.Start(handleRequest)
}
