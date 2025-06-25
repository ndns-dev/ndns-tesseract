package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	var messageBody customTypes.SQSMessageBody
	contentType := strings.ToLower(e.Headers["Content-Type"])

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		formData := utils.ParseFormURLEncoded(e.Body)
		messageBody.JobID = formData["jobId"]
		messageBody.ImageURL = formData["imageUrl"]
	} else {
		err := json.Unmarshal([]byte(e.Body), &messageBody)
		if err != nil {
			return utils.Response(utils.ErrorHandler(ctx, fmt.Errorf("could not unmarshal HTTP request body: %w", err), "", "", "HTTPRequestUnmarshal"))
		}
	}

	errResp, err := processOCRRequest(ctx, messageBody)
	return utils.Response(errResp, err)
}

func handleSQSEvent(ctx context.Context, e events.SQSEvent) (interface{}, error) {
	if len(e.Records) == 0 {
		return utils.ErrorHandler(ctx, fmt.Errorf("no records in SQS event"), "", "", "SQSEventHandler")
	}

	record := e.Records[0]
	var messageBody customTypes.SQSMessageBody
	err := json.Unmarshal([]byte(record.Body), &messageBody)
	if err != nil {
		return utils.ErrorHandler(ctx, fmt.Errorf("could not unmarshal SQS message body: %w", err), "", "", "SQSMessageUnmarshal")
	}

	return processOCRRequest(ctx, messageBody)
}

func processOCRRequest(ctx context.Context, messageBody customTypes.SQSMessageBody) (*customTypes.ErrorResponse, error) {
	var missingFields []string
	if messageBody.JobID == "" {
		missingFields = append(missingFields, "jobId")
	}
	if messageBody.ImageURL == "" {
		missingFields = append(missingFields, "imageUrl")
	}
	if len(missingFields) > 0 {
		return utils.ErrorHandler(
			ctx,
			fmt.Errorf("missing required fields: %s", strings.Join(missingFields, ", ")),
			messageBody.JobID,
			messageBody.ImageURL,
			"RequestValidation",
		)
	}

	log.Printf("Processing JobID: %s, ImageURL: %s", messageBody.JobID, messageBody.ImageURL)

	// --- OCR 수행을 위한 요청 바디 구성 ---
	ocrRequestPayload := customTypes.OCRRequest{
		ImageUrl: messageBody.ImageURL,
	}

	// OCRRequestPayload를 JSON 바이트 배열로 마샬링
	requestBodyBytes, err := json.Marshal(ocrRequestPayload)
	if err != nil {
		return utils.ErrorHandler(ctx, fmt.Errorf("failed to marshal OCR request payload: %w", err), messageBody.JobID, messageBody.ImageURL, "MarshalOCRRequest")
	}

	// OCR 수행
	ocrResultDetails, ocrErr := ocr.PerformOCR(requestBodyBytes)

	// 최종 결과를 저장할 customTypes.OCRJobDetails 객체 생성
	finalResult := customTypes.OCRJobDetails{
		JobID:     messageBody.JobID,
		ImageURL:  messageBody.ImageURL,
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

	// 1. OCR JobDetails 테이블 업데이트
	if err := utils.UpdateOCRJobDetails(ctx, finalResult); err != nil {
		return utils.ErrorHandler(ctx, fmt.Errorf("failed to update OCRJobDetails for JobID %s: %w", messageBody.JobID, err), messageBody.JobID, messageBody.ImageURL, "UpdateOCRJobDetails")
	}
	log.Printf("OCR JobDetails for JobID %s updated to status: %s with text: %s", messageBody.JobID, finalResult.Status, finalResult.OCRText)

	// 2. OCR 성공 시에만 캐싱 테이블에 저장
	if finalResult.Status == customTypes.JobStatusCompleted {
		ocrResult := customTypes.OCRResults{
			ImageURL:  finalResult.ImageURL,
			OCRText:   finalResult.OCRText,
			Status:    customTypes.OCRStatus(customTypes.JobStatusCompleted),
			Error:     finalResult.Error,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if err := utils.SaveOCRResult(ctx, ocrResult); err != nil {
			log.Printf("WARN: Failed to save OCR result to cache table for ImageURL %s (JobID: %s): %v", messageBody.ImageURL, messageBody.JobID, err)
		} else {
			log.Printf("OCR result for ImageURL %s (JobID: %s) saved to cache table.", messageBody.ImageURL, messageBody.JobID)
		}
	}

	// OCR이 실패했을 경우 에러 반환
	if finalResult.Status == customTypes.JobStatusFailed {
		return &customTypes.ErrorResponse{
			Message:   finalResult.Error,
			JobID:     finalResult.JobID,
			ImageURL:  finalResult.ImageURL,
			ErrorCode: "OCRExecutionFailed",
		}, fmt.Errorf(finalResult.Error)
	}

	// 성공 시 nil 반환
	return nil, nil
}

func main() {
	lambda.Start(handleRequest)
}
