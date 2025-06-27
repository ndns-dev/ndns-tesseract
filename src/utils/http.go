package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	customTypes "github.com/ndns-dev/ndns-tesseract/src/types"
)

func ParseFormURLEncoded(body string) map[string]string {
	result := make(map[string]string)
	values, err := url.ParseQuery(body)
	if err != nil {
		return result
	}
	for k, v := range values {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

func Response(data interface{}, err error) (interface{}, error) {
	if err != nil {
		if errResp, ok := data.(*customTypes.ErrorResponse); ok && errResp != nil {
			return &events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       fmt.Sprintf(`{"message": "%s", "jobId": "%s", "imageUrl": "%s", "errorCode": "%s"}`, errResp.Message, errResp.JobId, errResp.ImageURL, errResp.ErrorCode),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}, nil
		}
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"message": "%s"}`, err.Error()),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	var responseBody string
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return &events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"message": "Failed to marshal response: %s"}`, err.Error()),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}, nil
		}
		responseBody = string(jsonData)
	} else {
		responseBody = `{"message": "success"}`
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       responseBody,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func ErrorHandler(ctx context.Context, err error, jobId, imageURL, source string) (*customTypes.ErrorResponse, error) {
	errMsg := err.Error()
	log.Printf("FINAL ERROR in %s: %s (jobId: %s, ImageURL: %s)", source, errMsg, jobId, imageURL)
	NotifyErrorToWebhook(ctx, "ERROR", errMsg, jobId, imageURL, source)
	return &customTypes.ErrorResponse{
		Message:   errMsg,
		JobId:     jobId,
		ImageURL:  imageURL,
		ErrorCode: source,
	}, nil
}
