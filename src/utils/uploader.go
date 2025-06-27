package utils

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	customTypes "github.com/ndns-dev/ndns-tesseract/src/types"
)

var dbClient *dynamodb.Client

const OcrResultTableName = "OcrResult"
const OcrQueueStatusTableName = "OcrQueueStatus"

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config for DynamoDB in utils init: %v", err)
	}
	dbClient = dynamodb.NewFromConfig(cfg)

	log.Printf("Utils: DynamoDB client initialized.")
}

// SaveOcrResult: OCR 결과를 캐싱 테이블(OcrResults)에 저장
func SaveOcrResult(ctx context.Context, result customTypes.OcrResult) error {
	// ImageURL이 OcrResults 테이블의 Partition Key이므로 비어있으면 안 됩니다.
	if result.ImageUrl == "" {
		return fmt.Errorf("ImageURL cannot be empty when saving to OcrResultsCache table")
	}

	item, err := attributevalue.MarshalMap(result)
	if err != nil {
		return fmt.Errorf("failed to marshal cache item for ImageURL %s: %w", result.ImageUrl, err)
	}

	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(OcrResultTableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in OcrResultsCache DynamoDB table for ImageURL %s: %w", result.ImageUrl, err)
	}
	return nil
}

// UpdateOCRJobDetails: OCR Job 상세 테이블(OCRJobDetails)을 업데이트
func UpdateOCRJobDetails(ctx context.Context, result customTypes.OCRJobDetails) error {

	if result.JobId == "" {
		return fmt.Errorf("JobId cannot be empty when updating OCRJobDetails table")
	}

	item, err := attributevalue.MarshalMap(result)
	if err != nil {
		return fmt.Errorf("failed to marshal job details item for JobId %s: %w", result.JobId, err)
	}

	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(OcrQueueStatusTableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in OCRJobDetails DynamoDB table for JobId %s: %w", result.JobId, err)
	}
	return nil
}
