package main

import (
	"github.com/aws/aws-lambda-go/lambda"
)

func echo(input any) (any, error) {
	return input, nil
}

func main() {
	lambda.Start(echo)
}
