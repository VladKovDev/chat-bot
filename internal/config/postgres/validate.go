package database

import (
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres"
)

func Validate(cfg postgres.Config) error {
	if cfg.Host == "" {
		return fmt.Errorf("host is empty")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %v", cfg.Port)
	}

	if cfg.User == "" {
		return fmt.Errorf("user name is empty")
	}

	if cfg.Name == "" {
		return fmt.Errorf("cfg name is empty")
	}

	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if !validSSLModes[cfg.SSLMode] {
		return fmt.Errorf("sslmode must be (disable, require, verify-ca, verify-full), got: %v", cfg.SSLMode)
	}

	if cfg.MaxOpenConns < 1 {
		return fmt.Errorf("max open conns must be at least 1, got: %v", cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns < 0 {
		return fmt.Errorf("max idle conns must be non-negative, got: %v", cfg.MaxIdleConns)
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		return fmt.Errorf("max_idle_conns (%d) cannot exceed max_open_conns (%d)", cfg.MaxIdleConns, cfg.MaxOpenConns)
	}

	if cfg.ConnMaxLifetime < 0 {
		return fmt.Errorf("conn_max_lifetime must be positive, got %v", cfg.ConnMaxLifetime)
	}

	if cfg.ConnMaxIdleTime < 0 {
		return fmt.Errorf("conn_max_idle_time must be positive, got %v", cfg.ConnMaxIdleTime)
	}

	return nil
}