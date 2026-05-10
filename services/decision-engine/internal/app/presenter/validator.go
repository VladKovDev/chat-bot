package presenter

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// Validator validates response configuration
type Validator struct {
	responses map[string]*ResponseConfig
	logger    logger.Logger
}

// NewValidator creates a new response validator
func NewValidator(responses map[string]*ResponseConfig, logger logger.Logger) *Validator {
	return &Validator{
		responses: responses,
		logger:    logger,
	}
}

var (
	placeholderPattern = regexp.MustCompile(`#?\{([a-zA-Z0-9_]+)\}`)
	cyrillicPattern    = regexp.MustCompile(`[А-Яа-яЁё]`)
)

var expectedPlaceholdersByResponse = map[string][]string{
	"booking_found":           {"booking_number", "service", "master", "date", "time", "status"},
	"workspace_booking_found": {"booking_number", "workspace_type", "date", "time", "duration", "status"},
	"payment_found":           {"payment_id", "amount", "date", "status", "purpose"},
	"account_found":           {"user_id", "email", "phone", "status"},
	"escalation_context_sent": {"question"},
}

var requiredIntentKeys = []string{
	"greeting",
	"goodbye",
	"return_to_menu",
	"reset_conversation",
	"request_operator",
	"ask_booking_info",
	"ask_booking_status",
	"ask_cancellation_rules",
	"ask_reschedule_rules",
	"booking_not_found",
	"ask_workspace_info",
	"ask_workspace_prices",
	"ask_workspace_rules",
	"ask_workspace_status",
	"workspace_unavailable",
	"ask_payment_status",
	"payment_not_passed",
	"payment_not_activated",
	"ask_refund_rules",
	"ask_site_problem",
	"login_not_working",
	"code_not_received",
	"ask_account_help",
	"ask_account_status",
	"ask_services_info",
	"ask_prices",
	"ask_rules",
	"ask_location",
	"ask_faq",
	"report_complaint",
	"complaint_master",
	"complaint_premises",
	"general_question",
	"unknown",
}

var requiredIntentCategories = []string{
	"system",
	"operator",
	"booking",
	"workspace",
	"payment",
	"tech_issue",
	"account",
	"services",
	"complaint",
	"other",
	"fallback",
}

var validResolutionTypes = []string{
	"static_response",
	"operator_handoff",
	"knowledge",
	"business_lookup",
	"clarification",
	"fallback",
}

var validQuickReplyActions = []string{
	"send_text",
	"request_operator",
	"select_intent",
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

		for i, quickReply := range resp.QuickReplies {
			errors = append(errors, validateQuickReplyConfig(
				fmt.Sprintf("response_key: %s, quick_reply_index: %d", key, i),
				quickReply,
			)...)
		}

		placeholders, err := extractPlaceholders(resp.Message)
		if err != nil {
			errors = append(errors, fmt.Sprintf("invalid placeholders for response_key: %s: %v", key, err))
			continue
		}

		expectedPlaceholders, hasExpectation := expectedPlaceholdersByResponse[key]
		if len(placeholders) > 0 && !hasExpectation {
			errors = append(errors, fmt.Sprintf("unexpected placeholders for response_key: %s: %s", key, strings.Join(placeholders, ", ")))
			continue
		}
		if hasExpectation && !sameStringSet(placeholders, expectedPlaceholders) {
			errors = append(errors, fmt.Sprintf(
				"placeholder mismatch for response_key: %s: got [%s], want [%s]",
				key,
				strings.Join(placeholders, ", "),
				strings.Join(expectedPlaceholders, ", "),
			))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("response validation failed:\n%s", strings.Join(errors, "\n"))
	}

	v.logger.Info("response configuration validated successfully", v.logger.Int("responses", len(v.responses)))
	return nil
}

func (v *Validator) ValidateCatalog(catalog *IntentCatalog) error {
	if catalog == nil {
		return fmt.Errorf("intent catalog is nil")
	}
	if len(catalog.Intents) == 0 {
		return fmt.Errorf("intent catalog is empty")
	}

	errors := make([]string, 0)
	seenKeys := make(map[string]struct{}, len(catalog.Intents))
	seenCategories := make(map[string]struct{}, len(requiredIntentCategories))

	for _, intentDefinition := range catalog.Intents {
		if intentDefinition.Key == "" {
			errors = append(errors, "intent with empty key")
			continue
		}
		if _, exists := seenKeys[intentDefinition.Key]; exists {
			errors = append(errors, fmt.Sprintf("duplicate intent key: %s", intentDefinition.Key))
			continue
		}
		seenKeys[intentDefinition.Key] = struct{}{}
		if intentDefinition.Category != "" {
			seenCategories[intentDefinition.Category] = struct{}{}
		}

		errors = append(errors, v.validateIntentDefinition(intentDefinition)...)
	}

	for _, requiredIntentKey := range requiredIntentKeys {
		if _, ok := seenKeys[requiredIntentKey]; !ok {
			errors = append(errors, fmt.Sprintf("missing required intent: %s", requiredIntentKey))
		}
	}

	for _, requiredCategory := range requiredIntentCategories {
		if _, ok := seenCategories[requiredCategory]; !ok {
			errors = append(errors, fmt.Sprintf("missing required intent category: %s", requiredCategory))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("intent catalog validation failed:\n%s", strings.Join(errors, "\n"))
	}

	v.logger.Info("intent catalog validated successfully", v.logger.Int("intents", len(catalog.Intents)))
	return nil
}

func (v *Validator) validateIntentDefinition(intentDefinition IntentDefinition) []string {
	errors := make([]string, 0)

	if intentDefinition.Category == "" {
		errors = append(errors, fmt.Sprintf("intent %s has empty category", intentDefinition.Key))
	}
	if !slices.Contains(validResolutionTypes, intentDefinition.ResolutionType) {
		errors = append(errors, fmt.Sprintf("intent %s has unsupported resolution_type: %s", intentDefinition.Key, intentDefinition.ResolutionType))
	}
	if len(intentDefinition.Examples) < 8 {
		errors = append(errors, fmt.Sprintf("intent %s has %d examples, want at least 8", intentDefinition.Key, len(intentDefinition.Examples)))
	}
	if len(intentDefinition.E2ECoverage) == 0 {
		errors = append(errors, fmt.Sprintf("intent %s has no e2e coverage mapping", intentDefinition.Key))
	}

	for _, example := range intentDefinition.Examples {
		if strings.TrimSpace(example) == "" {
			errors = append(errors, fmt.Sprintf("intent %s contains an empty example", intentDefinition.Key))
			continue
		}
		if !cyrillicPattern.MatchString(example) {
			errors = append(errors, fmt.Sprintf("intent %s contains a non-Russian example: %s", intentDefinition.Key, example))
		}
	}

	switch intentDefinition.ResolutionType {
	case "static_response", "clarification", "fallback":
		errors = append(errors, v.requireResponseKey(intentDefinition.Key, "response_key", intentDefinition.ResponseKey)...)
	case "knowledge":
		errors = append(errors, v.requireResponseKey(intentDefinition.Key, "response_key", intentDefinition.ResponseKey)...)
		if strings.TrimSpace(intentDefinition.KnowledgeKey) == "" {
			errors = append(errors, fmt.Sprintf("intent %s is knowledge-based but knowledge_key is empty", intentDefinition.Key))
		}
	case "business_lookup":
		if strings.TrimSpace(intentDefinition.Action) == "" {
			errors = append(errors, fmt.Sprintf("intent %s is business_lookup but action is empty", intentDefinition.Key))
		} else if !slices.Contains(action.All(), intentDefinition.Action) {
			errors = append(errors, fmt.Sprintf("intent %s references unknown action: %s", intentDefinition.Key, intentDefinition.Action))
		}

		if strings.TrimSpace(intentDefinition.ResponseKey) == "" &&
			strings.TrimSpace(intentDefinition.FallbackResponseKey) == "" &&
			len(intentDefinition.ResultResponseKeys) == 0 {
			errors = append(errors, fmt.Sprintf("intent %s must define a response_key, fallback_response_key or result_response_keys", intentDefinition.Key))
		}

		errors = append(errors, v.optionalResponseKey(intentDefinition.Key, "response_key", intentDefinition.ResponseKey)...)
		errors = append(errors, v.optionalResponseKey(intentDefinition.Key, "fallback_response_key", intentDefinition.FallbackResponseKey)...)
		for _, responseKey := range intentDefinition.ResultResponseKeys {
			errors = append(errors, v.requireResponseKey(intentDefinition.Key, "result_response_key", responseKey)...)
		}
	case "operator_handoff":
		errors = append(errors, v.requireResponseKey(intentDefinition.Key, "response_key", intentDefinition.ResponseKey)...)
		if strings.TrimSpace(intentDefinition.Action) == "" {
			errors = append(errors, fmt.Sprintf("intent %s is operator_handoff but action is empty", intentDefinition.Key))
		} else if !slices.Contains(action.All(), intentDefinition.Action) {
			errors = append(errors, fmt.Sprintf("intent %s references unknown action: %s", intentDefinition.Key, intentDefinition.Action))
		}
	}

	for index, quickReply := range intentDefinition.QuickReplies {
		errors = append(errors, validateQuickReplyConfig(
			fmt.Sprintf("intent: %s, quick_reply_index: %d", intentDefinition.Key, index),
			quickReply,
		)...)
	}

	return errors
}

func (v *Validator) requireResponseKey(intentKey, fieldName, responseKey string) []string {
	if strings.TrimSpace(responseKey) == "" {
		return []string{fmt.Sprintf("intent %s has empty %s", intentKey, fieldName)}
	}
	return v.optionalResponseKey(intentKey, fieldName, responseKey)
}

func (v *Validator) optionalResponseKey(intentKey, fieldName, responseKey string) []string {
	if strings.TrimSpace(responseKey) == "" {
		return nil
	}
	if _, ok := v.responses[responseKey]; !ok {
		return []string{fmt.Sprintf("intent %s references missing %s: %s", intentKey, fieldName, responseKey)}
	}
	return nil
}

func validateQuickReplyConfig(scope string, quickReply QuickReplyConfig) []string {
	errors := make([]string, 0)
	if strings.TrimSpace(quickReply.ID) == "" {
		errors = append(errors, fmt.Sprintf("%s has empty quick reply id", scope))
	}
	if strings.TrimSpace(quickReply.Label) == "" {
		errors = append(errors, fmt.Sprintf("%s has empty quick reply label", scope))
	}
	if !slices.Contains(validQuickReplyActions, quickReply.Action) {
		errors = append(errors, fmt.Sprintf("%s has unsupported quick reply action: %s", scope, quickReply.Action))
	}

	switch quickReply.Action {
	case "send_text":
		if payloadText, _ := quickReply.Payload["text"].(string); strings.TrimSpace(payloadText) == "" {
			errors = append(errors, fmt.Sprintf("%s send_text quick reply must define payload.text", scope))
		}
	case "select_intent":
		if payloadIntent, _ := quickReply.Payload["intent"].(string); strings.TrimSpace(payloadIntent) == "" {
			errors = append(errors, fmt.Sprintf("%s select_intent quick reply must define payload.intent", scope))
		}
	}

	return errors
}

func extractPlaceholders(message string) ([]string, error) {
	matches := placeholderPattern.FindAllStringSubmatch(message, -1)
	placeholders := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if _, exists := seen[match[1]]; exists {
			continue
		}
		seen[match[1]] = struct{}{}
		placeholders = append(placeholders, match[1])
	}

	cleaned := placeholderPattern.ReplaceAllString(message, "")
	if strings.Contains(cleaned, "{") || strings.Contains(cleaned, "}") {
		return nil, fmt.Errorf("unbalanced placeholder braces")
	}

	return placeholders, nil
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	slices.Sort(leftCopy)
	slices.Sort(rightCopy)
	return slices.Equal(leftCopy, rightCopy)
}
