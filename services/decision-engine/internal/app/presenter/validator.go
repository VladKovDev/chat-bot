package presenter

import (
	"fmt"
	"strings"

	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// Validator validates response configuration
type Validator struct {
	responses map[string]*ResponseConfig
	logger    logger.Logger
}

// NewValidator creates a new response validator
func NewValidator(logger logger.Logger) *Validator {
	return &Validator{
		responses: make(map[string]*ResponseConfig),
		logger:    logger,
	}
}

// Validate checks the response configuration for structural errors
func (v *Validator) Validate() error {
	if len(v.responses) == 0 {
		return fmt.Errorf("response configuration is empty")
	}

	errors := make([]string, 0)

	// Check: All responses have valid structure
	for key, resp := range v.responses {
		if resp.Message == "" {
			errors = append(errors, fmt.Sprintf("empty message for response_key: %s", key))
		}

		// Options can be empty, but if provided, should not be empty strings
		for i, opt := range resp.Options {
			if opt == "" {
				errors = append(errors, fmt.Sprintf("empty option at index %d for response_key: %s", i, key))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("response validation failed:\n%s", strings.Join(errors, "\n"))
	}

	v.logger.Info("response configuration validated successfully", v.logger.Int("responses", len(v.responses)))
	return nil
}