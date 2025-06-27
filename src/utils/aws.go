package utils

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var (
	dynamoOnce   sync.Once
	dynamoClient *dynamodb.Client
)

// GetDynamoDBClient는 싱글톤 DynamoDB 클라이언트를 반환합니다.
func GetDynamoDBClient(ctx context.Context) *dynamodb.Client {
	dynamoOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			panic(err)
		}
		dynamoClient = dynamodb.NewFromConfig(cfg)
	})
	return dynamoClient
}
