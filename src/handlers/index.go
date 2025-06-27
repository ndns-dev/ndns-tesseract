package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
)

func HandleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// SQS 이벤트 체크
	var sqsEvent events.SQSEvent
	if err := json.Unmarshal(event, &sqsEvent); err == nil {
		if len(sqsEvent.Records) > 0 && sqsEvent.Records[0].EventSource == "aws:sqs" {
			return HandleSQSEvent(ctx, sqsEvent)
		}
	}

	// API Gateway 이벤트 체크
	var apiEvent events.APIGatewayProxyRequest
	if err := json.Unmarshal(event, &apiEvent); err == nil {
		if apiEvent.Body != "" {
			return HandleAPIGatewayEvent(ctx, apiEvent)
		}
	}

	log.Printf("Unsupported event type: %s", string(event))
	return nil, nil
}
