package main

import (
	"os"
	"strconv"
)

// NOTE: Зачем это?
type berCLIConfig struct {
	url string
}

type Config struct {
	// TODO: url string //K.I.S.S.
	berCLI           berCLIConfig
	expressionLength int
	workerPoolSize   int
}

// New returns a new Config struct.
// NOTE: why pointer?
func New() *Config {
	return &Config{
		berCLI: berCLIConfig{
			// TODO: "CALCULATOR_URL"
			url: getEnv("URL", "http://localhost:8080/?expr=")},
		expressionLength: getEnvAsInt("ExpressionLength", 10),
		workerPoolSize:   getEnvAsInt("WorkerPoolSize", 20),
	}
}

// getEnv reads an environment or return default value.
func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultValue
}

// getEnvAsInt reads an environment variable into integer or return a default value.
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}
