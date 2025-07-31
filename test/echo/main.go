package main

import (
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

func echo(input string) (string, error) {
	time.Sleep(500 * time.Millisecond)
	return input, nil
}

func main() {
	lambda.Start(echo)
}
