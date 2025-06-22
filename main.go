package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	ocr "github.com/ndns-dev/ndns-tesseract/src/ocr"
	customTypes "github.com/ndns-dev/ndns-tesseract/src/types"
	"github.com/ndns-dev/ndns-tesseract/src/utils"
)

func handleRequest(ctx context.Context, sqsEvent events.SQSEvent) (*customTypes.ErrorResponse, error) {
	// ---- 에러 발생 시 중앙 처리 헬퍼 함수 ----
	sendAndLogFinalError := func(err error, jobID, imageURL, source string) (*customTypes.ErrorResponse, error) {
		errMsg := err.Error()
		log.Printf("FINAL ERROR in %s: %s (JobID: %s, ImageURL: %s)", source, errMsg, jobID, imageURL)
		utils.NotifyErrorToWebhook(ctx, "ERROR", errMsg, jobID, imageURL, source)
		return &customTypes.ErrorResponse{
			Message:   errMsg,
			JobID:     jobID,
			ImageURL:  imageURL,
			ErrorCode: source,
		}, nil
	}
	// ----------------------------------------

	var jobID string                           // 모든 에러 처리 시 JobID를 전달하기 위해 미리 선언
	var imageURL string                        // 모든 에러 처리 시 ImageURL을 전달하기 위해 미리 선언
	var messageBody customTypes.SQSMessageBody // messageBody를 더 넓은 스코프에서 사용

	if len(sqsEvent.Records) == 0 {
		return sendAndLogFinalError(fmt.Errorf("no records in SQS event"), "", "", "SQSEventHandler")
	}

	record := sqsEvent.Records[0]

	err := json.Unmarshal([]byte(record.Body), &messageBody)
	if err != nil {
		return sendAndLogFinalError(fmt.Errorf("could not unmarshal SQS message body: %w", err), "", "", "SQSMessageUnmarshal")
	}

	// Unmarshal 성공 후 JobID와 ImageURL 설정 (에러 컨텍스트에 포함)
	jobID = messageBody.JobID
	imageURL = messageBody.ImageURL

	// JobID는 필수 확인 (Job당 이미지 1개로 가정)
	if jobID == "" {
		return sendAndLogFinalError(fmt.Errorf("SQS message body missing JobID"), "", imageURL, "SQSMessageValidation")
	}

	log.Printf("Processing JobID: %s, ImageURL: %s", jobID, imageURL)

	// --- OCR 수행을 위한 요청 바디 구성 ---
	ocrRequestPayload := customTypes.OCRRequest{
		ImageUrl: imageURL,
	}

	// OCRRequestPayload를 JSON 바이트 배열로 마샬링
	requestBodyBytes, err := json.Marshal(ocrRequestPayload)
	if err != nil {
		return sendAndLogFinalError(fmt.Errorf("failed to marshal OCR request payload: %w", err), jobID, imageURL, "MarshalOCRRequest")
	}
	// ------------------------------------

	// OCR 수행
	ocrResultDetails, ocrErr := ocr.PerformOCR(requestBodyBytes)

	// 최종 결과를 저장할 customTypes.OCRJobDetails 객체 생성
	finalResult := customTypes.OCRJobDetails{
		JobID:     jobID,
		ImageURL:  imageURL,
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
		return sendAndLogFinalError(fmt.Errorf("failed to update OCRJobDetails for JobID %s: %w", jobID, err), jobID, imageURL, "UpdateOCRJobDetails")
	}
	log.Printf("OCR JobDetails for JobID %s updated to status: %s", jobID, finalResult.Status)

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
			log.Printf("WARN: Failed to save OCR result to cache table for ImageURL %s (JobID: %s): %v", imageURL, jobID, err)
		} else {
			log.Printf("OCR result for ImageURL %s (JobID: %s) saved to cache table.", imageURL, jobID)
		}
	}

	// 성공 시 nil 반환
	return nil, nil
}

func main() {
	lambda.Start(handleRequest)
}
