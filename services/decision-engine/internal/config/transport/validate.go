package transport

import (
	"fmt"
	"time"

	"github.com/VladKovDev/chat-bot/internal/transport/http"
)

func Validate(cfg http.Config) error {
	if cfg.Address == "" {
		return fmt.Errorf("http address is empty")
	}

	if cfg.ReadTimeout < 0 {
		return fmt.Errorf("read_timeout must be non-negative, got: %v", cfg.ReadTimeout)
	}

	if cfg.ReadHeadTimeout < 0 {
		return fmt.Errorf("read_head_timeout must be non-negative, got: %v", cfg.ReadHeadTimeout)
	}

	if cfg.WriteTimeout < 0 {
		return fmt.Errorf("write_timeout must be non-negative, got: %v", cfg.WriteTimeout)
	}

	if cfg.IdleTimeout < 0 {
		return fmt.Errorf("idle_timeout must be non-negative, got: %v", cfg.IdleTimeout)
	}

	if cfg.MaxHeaderBytes < 0 {
		return fmt.Errorf("max_header_bytes must be non-negative, got: %v", cfg.MaxHeaderBytes)
	}

	if cfg.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative, got: %v", cfg.Timeout)
	}

	if cfg.BodyLimit < 0 {
		return fmt.Errorf("body_limit must be non-negative, got: %v", cfg.BodyLimit)
	}

	// Validate sensible minimums
	if cfg.ReadTimeout < time.Second {
		return fmt.Errorf("read_timeout should be at least 1s, got: %v", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout < time.Second {
		return fmt.Errorf("write_timeout should be at least 1s, got: %v", cfg.WriteTimeout)
	}

	return nil
}