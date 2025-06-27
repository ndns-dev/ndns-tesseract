package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/ndns-dev/ndns-tesseract/src/utils"
)

// HandleRequest는 Lambda 함수의 메인 핸들러입니다.
func HandleRequest(ctx context.Context, event interface{}) (interface{}, error) {
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
					return HandleAPIGatewayEvent(ctx, apiEvent)
				}
			}
		}
	}

	// SQS 이벤트 처리
	if sqsEvent, ok := event.(events.SQSEvent); ok {
		return HandleSQSEvent(ctx, sqsEvent)
	}

	return utils.Response(utils.ErrorHandler(ctx, fmt.Errorf("unsupported event type"), "", "", "EventTypeHandler"))
}
