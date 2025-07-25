package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	customTypes "github.com/ndns-dev/ndns-tesseract/src/types"
	"github.com/ndns-dev/ndns-tesseract/src/utils"
)

// HandleOcrWorkflow는 OCR 워크플로우 전체를 처리합니다.
func HandleOcrWorkflow(ctx context.Context, queueState customTypes.OcrQueueState) (*customTypes.OcrResult, error) {
	// 1. OCR 처리
	result, err := ProcessOcrRequest(ctx, queueState)
	if err != nil {
		return nil, err
	}

	// 2. 분석 API 호출
	analyzePayload := customTypes.AnalyzeCycleParam{
		Result: *result,
		State:  queueState,
	}
	jsonPayload, err := json.Marshal(analyzePayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal analyze payload: %w", err)
	}

	// API 요청 데이터 로깅
	log.Printf("Sending analyze API request with payload: %s", string(jsonPayload))

	apiUrl := os.Getenv("API_URL") + "/api/v1/search/analyze/cycle"
	log.Printf("Calling analyze API at URL: %s", apiUrl)

	resp, err := http.Post(apiUrl, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to call analyze API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 응답 본문 읽기
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed to read error response body: %v", err)
		} else {
			log.Printf("analyze API error response: %s", string(body))
		}
		log.Printf("analyze API returned non-200 status: %v, status code: %d", apiUrl, resp.StatusCode)
		return nil, fmt.Errorf("analyze API returned non-200 status: %d", resp.StatusCode)
	}

	// 응답 내용 로깅 (성공 시에도)
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("analyze API response: %s", string(respBody))

	// 3. DynamoDB에 저장
	dynamoClient := utils.GetDynamoDBClient(ctx)

	item, err := attributevalue.MarshalMap(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DynamoDB item: %w", err)
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(string(customTypes.OcrResultTableName)),
		Item:      item,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save to DynamoDB: %w", err)
	}

	return result, nil
}

// ProcessOcrRequest는 OCR 요청을 처리합니다.
func ProcessOcrRequest(ctx context.Context, queueState customTypes.OcrQueueState) (*customTypes.OcrResult, error) {
	// queueState 전체를 로깅
	jsonState, _ := json.MarshalIndent(queueState, "", "  ")
	log.Printf("Processing OCR request with queueState: %s", string(jsonState))

	// 필수 필드 검증
	if queueState.JobId == "" {
		return nil, fmt.Errorf("jobId is required")
	}
	if queueState.CrawlResult == nil {
		return nil, fmt.Errorf("crawlResult is required")
	}
	if queueState.CurrentPosition == "" {
		return nil, fmt.Errorf("currentPosition is required")
	}

	// CurrentPosition 유효성 검사
	validPositions := []customTypes.OcrPosition{
		customTypes.OcrPositionFirstImage,
		customTypes.OcrPositionFirstSticker,
		customTypes.OcrPositionSecondSticker,
		customTypes.OcrPositionLastImage,
		customTypes.OcrPositionLastSticker,
	}
	isValidPosition := false
	for _, pos := range validPositions {
		if queueState.CurrentPosition == pos {
			isValidPosition = true
			break
		}
	}
	if !isValidPosition {
		return nil, fmt.Errorf("invalid currentPosition: %s", queueState.CurrentPosition)
	}

	// 이미지 URL 가져오기
	log.Printf("Attempting to get image URL for position: %s", queueState.CurrentPosition)
	log.Printf("CrawlResult details: %+v", queueState.CrawlResult)
	imageUrl := queueState.CrawlResult.GetImageUrlByPosition(queueState.CurrentPosition)
	if imageUrl == "" {
		log.Printf("Failed to get image URL. CrawlResult: %+v, Position: %s", queueState.CrawlResult, queueState.CurrentPosition)
		return nil, fmt.Errorf("no image URL found for position: %s", queueState.CurrentPosition)
	}
	log.Printf("Successfully got image URL: %s", imageUrl)

	OcrResult, err := utils.PerformOCR(imageUrl)
	if err != nil {
		return nil, err
	}

	// OCR 결과 생성
	result := &customTypes.OcrResult{
		ImageUrl:    imageUrl,
		JobId:       queueState.JobId,
		OcrText:     OcrResult,
		Position:    queueState.CurrentPosition,
		ProcessedAt: time.Now(),
		Error:       "",
	}

	return result, nil
}
