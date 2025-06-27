package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/ndns-dev/ndns-tesseract/src/handlers"
)

func main() {
	lambda.Start(handlers.HandleRequest)
}
