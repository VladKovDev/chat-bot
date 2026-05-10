package actions

import (
	"context"

	appprovider "github.com/VladKovDev/chat-bot/internal/app/provider"
	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindUserAccount finds accounts through the configured account provider.
type FindUserAccount struct {
	logger   logger.Logger
	provider appprovider.AccountProvider
}

// NewFindUserAccount creates a new FindUserAccount action
func NewFindUserAccount(logger logger.Logger, datasets ...*appseed.Dataset) *FindUserAccount {
	var dataset *appseed.Dataset
	if len(datasets) > 0 {
		dataset = datasets[0]
	}

	return &FindUserAccount{
		logger:   logger,
		provider: appprovider.NewMockAccountService(dataset),
	}
}

func (a *FindUserAccount) Execute(ctx context.Context, data action.ActionData) error {
	identifier, identifierType := providerIdentifier(data)
	response, audit, err := a.provider.LookupAccount(ctx, appprovider.AccountLookupRequest{
		Identifier:     identifier,
		IdentifierType: identifierType,
	})
	result := accountActionResult(response, audit)
	if err != nil {
		result = safeProviderErrorResult(audit, err)
	}

	storeProviderOutcome(data, "user_account_info", result, audit)

	a.logger.Info("MOCK: find_user_account executed",
		a.logger.String("identifier_hash", observability.HashForLog(identifier)),
		a.logger.Int("identifier_length", observability.LenForLog(identifier)),
		a.logger.String("provider", audit.Provider),
		a.logger.String("source", audit.Source),
		a.logger.String("status", audit.Status),
		a.logger.Int64("duration_ms", audit.DurationMS),
		a.logger.String("error_code", audit.ErrorCode))

	return nil
}

func accountActionResult(response appprovider.AccountLookupResponse, audit appprovider.ActionAudit) map[string]any {
	result := map[string]any{
		"status": audit.Status,
		"found":  response.Found,
		"source": response.Source,
	}
	addProviderErrorCode(result, audit)
	if response.AccountID != "" {
		result["user_id"] = response.AccountID
		result["account_id"] = response.AccountID
	}
	if response.Email != "" {
		result["email"] = response.Email
	}
	if response.Phone != "" {
		result["phone"] = response.Phone
	}
	if response.Status != "" {
		result["account_status"] = response.Status
	}
	return result
}
