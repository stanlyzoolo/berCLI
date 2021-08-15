package main

import (
	"os"
)

type berCLIConfig struct {
	url              string
	expressionLength string
	workerPoolSize   string
}

type Config struct {
	berCLI berCLIConfig
}

// New returns a new Config struct.
func NewConfig() *Config {
	return &Config{
		berCLI: berCLIConfig{
			url:              getEnv("URL", "http://localhost:8080/?expr="),
			expressionLength: getEnv("ExpressionLength", "10"),
			workerPoolSize:   getEnv("WorkerPoolSize", "20"),
		},
	}
}

// getEnv reads an environment or return default value.
func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultValue
}
