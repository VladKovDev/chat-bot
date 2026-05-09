package actions

import (
	"context"
	"fmt"
	"regexp"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// ValidateIdentifier validates identifier format before querying DB
type ValidateIdentifier struct {
	logger logger.Logger
}

// NewValidateIdentifier creates a new ValidateIdentifier action
func NewValidateIdentifier(logger logger.Logger) *ValidateIdentifier {
	return &ValidateIdentifier{logger: logger}
}

// Execute validates identifier format
func (a *ValidateIdentifier) Execute(ctx context.Context, data action.ActionData) error {
	// Extract identifier
	identifier, _ := data.Context["provided_identifier"].(string)
	if identifier == "" {
		identifier = data.UserText
	}

	// Determine identifier type from context
	idType, _ := data.Context["identifier_type"].(string)
	if idType == "" {
		// Auto-detect identifier type
		idType = a.detectIdentifierType(identifier)
	}

	// Validate format
	validationResult := a.validateFormat(identifier, idType)

	// Store result
	data.Context["action_result"] = validationResult
	data.Context["identifier_validation"] = validationResult

	// Log validation
	a.logger.Debug("identifier validated",
		a.logger.String("identifier", identifier),
		a.logger.String("type", idType),
		a.logger.Bool("valid", validationResult["valid"].(bool)))

	// Return error if invalid (this will stop execution chain)
	if !validationResult["valid"].(bool) {
		return fmt.Errorf("invalid %s format: %s", idType, identifier)
	}

	return nil
}

// detectIdentifierType auto-detects identifier type from input
func (a *ValidateIdentifier) detectIdentifierType(identifier string) string {
	// Check for booking number pattern (БРГ-XXXXXX)
	if matched, _ := regexp.MatchString(`^БРГ-\d{6}$`, identifier); matched {
		return "booking_number"
	}

	// Check for workspace booking pattern (WRK-XXX-XXX)
	if matched, _ := regexp.MatchString(`^WRK-(HOT|FIX|OFC1|OFC4)-\d{3}$`, identifier); matched {
		return "workspace_booking"
	}

	// Check for payment ID pattern (PAY-XXXXXX)
	if matched, _ := regexp.MatchString(`^PAY-\d{6}$`, identifier); matched {
		return "payment_id"
	}

	// Check for user ID pattern (usr-XXXXXX)
	if matched, _ := regexp.MatchString(`^usr-\d{6}$`, identifier); matched {
		return "user_id"
	}

	// Check for phone pattern
	if matched, _ := regexp.MatchString(`^\+7 \(\d{3}\) \d{3}-\d{2}-\d{2}$`, identifier); matched {
		return "phone"
	}

	// Check for simple phone (10 digits)
	if matched, _ := regexp.MatchString(`^\d{10}$`, identifier); matched {
		return "phone"
	}

	// Check for email pattern
	if matched, _ := regexp.MatchString(`^[^@]+@[^@]+\.[^@]+$`, identifier); matched {
		return "email"
	}

	// Default: unknown
	return "unknown"
}

// validateFormat validates identifier format based on type
func (a *ValidateIdentifier) validateFormat(identifier, idType string) map[string]interface{} {
	result := map[string]interface{}{
		"valid":          false,
		"identifier":     identifier,
		"identifier_type": idType,
	}

	switch idType {
	case "booking_number":
		pattern := regexp.MustCompile(`^БРГ-\d{6}$`)
		if pattern.MatchString(identifier) {
			result["valid"] = true
			result["normalized"] = identifier
		}

	case "workspace_booking":
		pattern := regexp.MustCompile(`^WRK-(HOT|FIX|OFC1|OFC4)-\d{3}$`)
		if pattern.MatchString(identifier) {
			result["valid"] = true
			result["normalized"] = identifier
		}

	case "payment_id":
		pattern := regexp.MustCompile(`^PAY-\d{6}$`)
		if pattern.MatchString(identifier) {
			result["valid"] = true
			result["normalized"] = identifier
		}

	case "user_id":
		pattern := regexp.MustCompile(`^usr-\d{6}$`)
		if pattern.MatchString(identifier) {
			result["valid"] = true
			result["normalized"] = identifier
		}

	case "phone":
		// Check for formatted phone: +7 (XXX) XXX-XX-XX
		formattedPattern := regexp.MustCompile(`^\+7 \(\d{3}\) \d{3}-\d{2}-\d{2}$`)
		// Check for simple phone: 10 digits
		simplePattern := regexp.MustCompile(`^\d{10}$`)

		if formattedPattern.MatchString(identifier) {
			result["valid"] = true
			result["normalized"] = identifier
		} else if simplePattern.MatchString(identifier) {
			result["valid"] = true
			// Normalize to formatted format
			result["normalized"] = fmt.Sprintf("+7 (%s) %s-%s-%s",
				identifier[0:3], identifier[3:6], identifier[6:8], identifier[8:10])
		}

	case "email":
		// Basic email validation
		emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if emailPattern.MatchString(identifier) {
			result["valid"] = true
			result["normalized"] = identifier
		}

	default:
		// Unknown type - mark as invalid
		result["valid"] = false
		result["error"] = "unknown identifier type"
	}

	if result["valid"] == false {
		if _, exists := result["error"]; !exists {
			result["error"] = fmt.Sprintf("invalid %s format", idType)
		}
	}

	return result
}
