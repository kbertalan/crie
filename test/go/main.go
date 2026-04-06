package main

import (
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

func echo(input any) (any, error) {
	time.Sleep(100 * time.Millisecond)
	return input, nil
}

func main() {
	lambda.Start(echo)
}
