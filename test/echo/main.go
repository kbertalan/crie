package main

import (
	"github.com/aws/aws-lambda-go/lambda"
)

func echo(input string) (string, error) {
	return input, nil
}

func main() {
	lambda.Start(echo)
}
