package nlp

import (
	"fmt"
	"strings"

	infranlp "github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
)

func Validate(cfg infranlp.EmbedderConfig) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("nlp base_url is required")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("nlp timeout must be positive")
	}
	if cfg.ExpectedDimension <= 0 {
		return fmt.Errorf("nlp expected_dimension must be positive")
	}
	return nil
}
