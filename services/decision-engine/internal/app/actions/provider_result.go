package actions

import (
	"errors"

	appprovider "github.com/VladKovDev/chat-bot/internal/app/provider"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

func providerIdentifier(data action.ActionData) (string, string) {
	var identifier, identifierType string
	if data.Context != nil {
		identifier, _ = data.Context["provided_identifier"].(string)
		identifierType, _ = data.Context["identifier_type"].(string)
	}
	if identifier == "" {
		identifier = data.UserText
	}
	return identifier, identifierType
}

func storeProviderOutcome(data action.ActionData, metadataKey string, result map[string]any, audit appprovider.ActionAudit) {
	if data.Context == nil {
		data.Context = map[string]any{}
	}
	data.Context["action_result"] = result
	data.Context["action_audit"] = actionAuditMap(audit)

	if data.Session != nil {
		if data.Session.Metadata == nil {
			data.Session.Metadata = map[string]any{}
		}
		data.Session.Metadata[metadataKey] = result
	}
}

func safeProviderErrorResult(audit appprovider.ActionAudit, err error) map[string]any {
	code := audit.ErrorCode
	var providerErr *appprovider.Error
	if errors.As(err, &providerErr) && providerErr.Code != "" {
		code = providerErr.Code
	}
	if code == "" {
		code = appprovider.CodeProviderUnavailable
	}

	return map[string]any{
		"status":     audit.Status,
		"found":      false,
		"source":     audit.Source,
		"error_code": code,
	}
}

func actionAuditMap(audit appprovider.ActionAudit) map[string]any {
	result := map[string]any{
		"provider":    audit.Provider,
		"source":      audit.Source,
		"status":      audit.Status,
		"duration_ms": audit.DurationMS,
	}
	if audit.ErrorCode != "" {
		result["error_code"] = audit.ErrorCode
	}
	return result
}

func addProviderErrorCode(result map[string]any, audit appprovider.ActionAudit) {
	if audit.ErrorCode != "" {
		result["error_code"] = audit.ErrorCode
	}
}
