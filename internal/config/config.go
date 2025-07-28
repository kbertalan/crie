package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	ProgramName                     string
	CommandName                     string
	CommandArgs                     []string
	OriginalEnvironment             []string
	OriginalAWSLambdaRuntimeAPI     string
	MaxConcurrency                  uint32
	InitialConcurrency              uint32
	QueueSize                       int
	WaitForQueueCapacity            time.Duration
	ServerAddress                   ListenAddress
	ServerShutdownTimeout           time.Duration
	LambdaName                      string
	MaxHandleAttempts               uint32
	DelayBetweenHandleAttempts      time.Duration
	RAPIServerShutdownTimeout       time.Duration
	LambdaRuntimeDeadline           time.Duration
	LambdaRuntimeInvokedFunctionArn string
}

const (
	AWS_LAMBDA_RUNTIME_API                   = "AWS_LAMBDA_RUNTIME_API"
	CRIE_MAX_CONCURRENCY                     = "CRIE_MAX_CONCURRENCY"
	CRIE_INITIAL_CONCURRENCY                 = "CRIE_INITIAL_CONCURRENCY"
	CRIE_QUEUE_SIZE                          = "CRIE_QUEUE_SIZE"
	CRIE_WAIT_FOR_QUEUE_CAPACITY             = "CRIE_WAIT_FOR_QUEUE_CAPACITY"
	CRIE_SERVER_ADDRESS                      = "CRIE_SERVER_ADDRESS"
	CRIE_SERVER_SHUTDOWN_TIMEOUT             = "CRIE_SERVER_SHUTDOWN_TIMEOUT"
	CRIE_LAMBDA_NAME                         = "CRIE_LAMBDA_NAME"
	CRIE_MAX_HANDLE_ATTEMPTS                 = "CRIE_MAX_HANDLE_ATTEMPTS"
	CRIE_DELAY_BETWEEN_HANDLE_ATTEMPTS       = "CRIE_DELAY_BETWEEN_HANDLE_ATTEMPTS"
	CRIE_RAPI_SERVER_SHUTDOWN_TIMEOUT        = "CRIE_RAPI_SERVER_SHUTDOWN_TIMEOUT"
	CRIE_LAMBDA_RUNTIME_DEADLINE             = "CRIE_LAMBDA_RUNTIME_DEADLINE"
	CRIE_LAMBDA_RUNTIME_INVOKED_FUNCTION_ARN = "CRIE_LAMBDA_RUNTIME_INVOKED_FUNCTION_ARN"

	defaultMaxConcurrency                 uint32        = 2
	defaultInitialConcurrency             uint32        = 1
	defaultQueueSize                      int           = 1000
	defaultWaitForQueueCapacity           time.Duration = 100 * time.Millisecond
	defaultServerAddress                  ListenAddress = ":10000"
	defaultServerShutdownTimeout          time.Duration = 10 * time.Second
	defaultLambdaName                                   = "function"
	defaultMaxHandleAttempts              uint32        = 100
	defaultDelayBetweenHandleAttempts     time.Duration = 100 * time.Millisecond
	defaultRAPIServerShutdownTimeout      time.Duration = 9 * time.Second
	defaultLambdaRuntimeDeadline          time.Duration = 90 * time.Second
	defaultLambdaRuntimeInvokedFuntionArn string        = "arn:aws:lambda:us-east-2:123456789012:function:custom-runtime"
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

	cfg.InitialConcurrency, err = parseEnvUint32(CRIE_INITIAL_CONCURRENCY, defaultInitialConcurrency)
	if err != nil {
		return cfg, err
	}

	cfg.QueueSize, err = parseEnvInt(CRIE_QUEUE_SIZE, defaultQueueSize)
	if err != nil {
		return cfg, err
	}

	cfg.WaitForQueueCapacity, err = parseEnv(CRIE_WAIT_FOR_QUEUE_CAPACITY, defaultWaitForQueueCapacity, time.ParseDuration)
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

	cfg.MaxHandleAttempts, err = parseEnvUint32(CRIE_MAX_HANDLE_ATTEMPTS, defaultMaxHandleAttempts)
	if err != nil {
		return cfg, err
	}

	cfg.DelayBetweenHandleAttempts, err = parseEnv(CRIE_DELAY_BETWEEN_HANDLE_ATTEMPTS, defaultDelayBetweenHandleAttempts, time.ParseDuration)
	if err != nil {
		return cfg, err
	}

	cfg.RAPIServerShutdownTimeout, err = parseEnv(CRIE_RAPI_SERVER_SHUTDOWN_TIMEOUT, defaultRAPIServerShutdownTimeout, time.ParseDuration)
	if err != nil {
		return cfg, err
	}

	if cfg.RAPIServerShutdownTimeout >= cfg.ServerShutdownTimeout {
		return cfg, fmt.Errorf("rapi.server shutdown timeout (%s) must be lower than server shutdown timeout (%s)", cfg.RAPIServerShutdownTimeout, cfg.ServerShutdownTimeout)
	}

	cfg.LambdaRuntimeDeadline, err = parseEnv(CRIE_LAMBDA_RUNTIME_DEADLINE, defaultLambdaRuntimeDeadline, time.ParseDuration)
	if err != nil {
		return cfg, err
	}

	if cfg.LambdaRuntimeDeadline > 900*time.Second {
		return cfg, fmt.Errorf("lambda runtime deadline cannot be higher than 15 minutes, but it was %s", cfg.LambdaRuntimeDeadline)
	}

	cfg.LambdaRuntimeInvokedFunctionArn = getEnv(CRIE_LAMBDA_RUNTIME_INVOKED_FUNCTION_ARN, defaultLambdaRuntimeInvokedFuntionArn)

	return cfg, nil
}
