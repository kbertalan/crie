package config

import (
	"fmt"
	"net"
	"strconv"
)

type ListenAddress string

func (a ListenAddress) ProcessAddress(i int) ListenAddress {
	_, portStr, err := net.SplitHostPort(string(a))
	if err != nil {
		panic(fmt.Sprintf("could not extract host and port from %s", a))
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("could not convert port string to int from %s", portStr))
	}

	return ListenAddress(fmt.Sprintf(":%d", port+i+1))
}

func (a ListenAddress) AwsLambdaRuntimeAPI() string {
	_, portStr, err := net.SplitHostPort(string(a))
	if err != nil {
		panic(fmt.Sprintf("could not extract host and port from %s", a))
	}

	return fmt.Sprintf("localhost:%s", portStr)
}
