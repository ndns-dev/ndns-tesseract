package utils

import (
	"context"
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

func Response(errResp *customTypes.ErrorResponse, err error) (interface{}, error) {
	if err != nil {
		if errResp != nil {
			return &events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       fmt.Sprintf(`{"message": "%s", "jobId": "%s", "imageUrl": "%s", "errorCode": "%s"}`, errResp.Message, errResp.JobID, errResp.ImageURL, errResp.ErrorCode),
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

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"message": "success"}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func ErrorHandler(ctx context.Context, err error, jobID, imageURL, source string) (*customTypes.ErrorResponse, error) {
	errMsg := err.Error()
	log.Printf("FINAL ERROR in %s: %s (JobID: %s, ImageURL: %s)", source, errMsg, jobID, imageURL)
	NotifyErrorToWebhook(ctx, "ERROR", errMsg, jobID, imageURL, source)
	return &customTypes.ErrorResponse{
		Message:   errMsg,
		JobID:     jobID,
		ImageURL:  imageURL,
		ErrorCode: source,
	}, nil
}
