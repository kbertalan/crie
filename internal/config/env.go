package config

import (
	"net"
	"os"
	"strconv"
)

func getEnv(key string, defaultValue string) string {
	if value, found := os.LookupEnv(key); found {
		return value
	}

	return defaultValue
}

func parseEnv[T any](key string, defaultValue T, parserFn func(string) (T, error)) (T, error) {
	valueStr, found := os.LookupEnv(key)
	if !found {
		return defaultValue, nil
	}

	return parserFn(valueStr)
}

func parseEnvUint32(key string, defaultValue uint32) (uint32, error) {
	return parseEnv(key, defaultValue, func(valueStr string) (uint32, error) {
		value, err := strconv.ParseUint(valueStr, 10, 32)
		if err != nil {
			return 0, err
		}

		return uint32(value), nil
	})
}

func parseEnvInt(key string, defaultValue int) (int, error) {
	return parseEnv(key, defaultValue, func(valueStr string) (int, error) {
		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return 0, err
		}

		return int(value), nil
	})
}

func parseEnvListenAddress(key string, defaultValue ListenAddress) (ListenAddress, error) {
	return parseEnv(key, defaultValue, func(valueStr string) (ListenAddress, error) {
		_, _, err := net.SplitHostPort(valueStr)
		if err != nil {
			return "", err
		}

		return ListenAddress(valueStr), nil
	})
}
