package config

import (
	"fmt"
)

// Validate validates the configuration
func Validate(cfg Config) error {
	// Decision Engine
	if cfg.DecisionEngine.URL == "" {
		return fmt.Errorf("decision_engine.url is required")
	}
	if cfg.DecisionEngine.Timeout <= 0 {
		return fmt.Errorf("decision_engine.timeout must be positive, got: %v", cfg.DecisionEngine.Timeout)
	}

	// Server
	if cfg.Server.Address == "" {
		return fmt.Errorf("server.address is required")
	}
	if cfg.Server.ReadBufferSize <= 0 {
		return fmt.Errorf("server.read_buffer_size must be positive, got: %d", cfg.Server.ReadBufferSize)
	}
	if cfg.Server.WriteBufferSize <= 0 {
		return fmt.Errorf("server.write_buffer_size must be positive, got: %d", cfg.Server.WriteBufferSize)
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		return fmt.Errorf("server.allowed_origins must contain at least one origin")
	}
	for _, origin := range cfg.Server.AllowedOrigins {
		if origin == "" {
			return fmt.Errorf("server.allowed_origins must not contain empty values")
		}
	}

	// Log
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[cfg.Log.Level] {
		return fmt.Errorf("log.level must be one of: debug, info, warn, error, got: %s", cfg.Log.Level)
	}
	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[cfg.Log.Format] {
		return fmt.Errorf("log.format must be one of: json, text, got: %s", cfg.Log.Format)
	}

	return nil
}
