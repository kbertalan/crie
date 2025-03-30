package config

import (
	"errors"
	"os"
	"time"
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
	ServerShutdownTimeout       time.Duration
	LambdaName                  string
}

const (
	AWS_LAMBDA_RUNTIME_API       = "AWS_LAMBDA_RUNTIME_API"
	CRIE_MAX_CONCURRENCY         = "CRIE_MAX_CONCURRENCY"
	CRIE_SERVER_ADDRESS          = "CRIE_SERVER_ADDRESS"
	CRIE_SERVER_SHUTDOWN_TIMEOUT = "CRIE_SERVER_SHUTDOWN_TIMEOUT"
	CRIE_LAMBDA_NAME             = "CRIE_LAMBDA_NAME"

	defaultMaxConcurrency        uint32        = 2
	defaultServerAddress         ListenAddress = ":10000"
	defaultServerShutdownTimeout time.Duration = 10 * time.Second
	defaultLambdaName                          = "function"
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

	var err error
	cfg.MaxConcurrency, err = parseEnvUint32(CRIE_MAX_CONCURRENCY, defaultMaxConcurrency)
	if err != nil {
		return cfg, err
	}

	cfg.ServerAddress, err = parseEnvListenAddress(CRIE_SERVER_ADDRESS, defaultServerAddress)
	if err != nil {
		return cfg, err
	}

	cfg.ServerShutdownTimeout, err = parseEnv(CRIE_SERVER_SHUTDOWN_TIMEOUT, defaultServerShutdownTimeout, time.ParseDuration)
	if err != nil {
		return cfg, err
	}

	cfg.LambdaName = getEnv(CRIE_LAMBDA_NAME, defaultLambdaName)

	return cfg, nil
}
