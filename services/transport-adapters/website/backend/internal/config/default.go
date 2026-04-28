package config

import "time"

// SetDefaultConfig returns the default configuration
func SetDefaultConfig() Config {
	return Config{
		DecisionEngine: DecisionEngine{
			URL:     "http://localhost:8080",
			Timeout: 10 * time.Second,
		},
		Server: Server{
			Address:         ":8081",
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		Log: Log{
			Level:  "info",
			Format: "json",
		},
	}
}