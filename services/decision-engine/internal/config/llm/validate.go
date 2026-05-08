package llm

import (
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/infrastructure/llm"
)

func Validate(cfg llm.Config) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("base_url is empty")
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got: %v", cfg.Timeout)
	}

	if cfg.CBMaxRequests < 1 {
		return fmt.Errorf("cb_max_requests must be at least 1, got: %v", cfg.CBMaxRequests)
	}

	if cfg.CBInterval <= 0 {
		return fmt.Errorf("cb_interval must be positive, got: %v", cfg.CBInterval)
	}

	if cfg.CBMaxFailures < 1 {
		return fmt.Errorf("cb_max_failures must be at least 1, got: %v", cfg.CBMaxFailures)
	}

	if cfg.CBTimeout <= 0 {
		return fmt.Errorf("cb_timeout must be positive, got: %v", cfg.CBTimeout)
	}

	return nil
}