package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type ListenAddress string

type Config struct {
	ProgramName                 string
	CommandName                 string
	CommandArgs                 []string
	OriginalEnvironment         []string
	OriginalAWSLambdaRuntimeAPI string
	MaxConcurrency              uint32
	ServerAddress               ListenAddress
}

const (
	AWS_LAMBDA_RUNTIME_API = "AWS_LAMBDA_RUNTIME_API"
	CRIE_MAX_CONCURRENCY   = "CRIE_MAX_CONCURRENCY"
	CRIE_SERVER_ADDRESS    = "CRIE_SERVER_ADDRESS"

	defaultMaxConcurrency uint32        = 2
	defaultServerAddress  ListenAddress = ":10000"
)

func Detect() (Config, error) {
	var cfg Config
	if len(os.Args) < 2 {
		return cfg, errors.New("not enough parameters")
	}

	cfg.ProgramName = os.Args[0]
	cfg.CommandName = os.Args[1]
	cfg.CommandArgs = os.Args[2:]

	cfg.OriginalEnvironment = os.Environ()

	cfg.OriginalAWSLambdaRuntimeAPI, _ = os.LookupEnv(AWS_LAMBDA_RUNTIME_API)

	cfg.MaxConcurrency = defaultMaxConcurrency
	if maxConcurrencyStr, found := os.LookupEnv(CRIE_MAX_CONCURRENCY); found {
		if maxConcurrency, err := strconv.ParseUint(maxConcurrencyStr, 10, 32); err != nil {
			return cfg, fmt.Errorf("unable to parse %s: %w", CRIE_MAX_CONCURRENCY, err)
		} else {
			cfg.MaxConcurrency = uint32(maxConcurrency)
		}
	}

	var err error
	cfg.MaxConcurrency, err = parseEnvUint32(CRIE_MAX_CONCURRENCY, defaultMaxConcurrency)
	if err != nil {
		return cfg, err
	}

	cfg.ServerAddress, err = parseEnvListenAddress(CRIE_SERVER_ADDRESS, defaultServerAddress)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
