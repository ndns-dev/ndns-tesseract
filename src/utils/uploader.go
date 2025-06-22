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

const ocrResultsTableName = "OCRResults"
const ocrJobDetailsTableName = "OCRJobDetails"

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config for DynamoDB in utils init: %v", err)
	}
	dbClient = dynamodb.NewFromConfig(cfg)

	log.Printf("Utils: DynamoDB client initialized.")
}

// SaveOCRResult: OCR 결과를 캐싱 테이블(OCRResults)에 저장
func SaveOCRResult(ctx context.Context, result customTypes.OCRResults) error {
	// ImageURL이 OCRResults 테이블의 Partition Key이므로 비어있으면 안 됩니다.
	if result.ImageURL == "" {
		return fmt.Errorf("ImageURL cannot be empty when saving to OCRResultsCache table")
	}

	item, err := attributevalue.MarshalMap(result)
	if err != nil {
		return fmt.Errorf("failed to marshal cache item for ImageURL %s: %w", result.ImageURL, err)
	}

	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(ocrResultsTableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in OCRResultsCache DynamoDB table for ImageURL %s: %w", result.ImageURL, err)
	}
	return nil
}

// UpdateOCRJobDetails: OCR Job 상세 테이블(OCRJobDetails)을 업데이트
func UpdateOCRJobDetails(ctx context.Context, result customTypes.OCRJobDetails) error {

	if result.JobID == "" {
		return fmt.Errorf("JobID cannot be empty when updating OCRJobDetails table")
	}

	item, err := attributevalue.MarshalMap(result)
	if err != nil {
		return fmt.Errorf("failed to marshal job details item for JobID %s: %w", result.JobID, err)
	}

	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(ocrJobDetailsTableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in OCRJobDetails DynamoDB table for JobID %s: %w", result.JobID, err)
	}
	return nil
}
