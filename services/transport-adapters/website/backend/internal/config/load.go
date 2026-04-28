package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration from viper
func LoadConfig(v *viper.Viper) (Config, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()

	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	// Decision Engine
	_ = v.BindEnv("decision_engine.url", "DECISION_ENGINE_URL")
	_ = v.BindEnv("decision_engine.timeout", "DECISION_ENGINE_TIMEOUT")

	// Server
	_ = v.BindEnv("server.address", "SERVER_ADDRESS")
	_ = v.BindEnv("server.read_buffer_size", "WS_READ_BUFFER_SIZE")
	_ = v.BindEnv("server.write_buffer_size", "WS_WRITE_BUFFER_SIZE")

	// Log
	_ = v.BindEnv("log.level", "LOG_LEVEL")
	_ = v.BindEnv("log.format", "LOG_FORMAT")
}