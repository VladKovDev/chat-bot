package actions

import (
	"context"

	appprovider "github.com/VladKovDev/chat-bot/internal/app/provider"
	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindPayment finds payment records through the configured payment provider.
type FindPayment struct {
	logger   logger.Logger
	provider appprovider.PaymentProvider
}

// NewFindPayment creates a new FindPayment action
func NewFindPayment(logger logger.Logger, datasets ...*appseed.Dataset) *FindPayment {
	var dataset *appseed.Dataset
	if len(datasets) > 0 {
		dataset = datasets[0]
	}

	return &FindPayment{
		logger:   logger,
		provider: appprovider.NewMockPaymentService(dataset),
	}
}

func (a *FindPayment) Execute(ctx context.Context, data action.ActionData) error {
	identifier, identifierType := providerIdentifier(data)
	response, audit, err := a.provider.LookupPayment(ctx, appprovider.PaymentLookupRequest{
		Identifier:     identifier,
		IdentifierType: identifierType,
	})
	result := paymentActionResult(response, audit)
	if err != nil {
		result = safeProviderErrorResult(audit, err)
	}

	storeProviderOutcome(data, "payment_info", result, audit)

	a.logger.Info("MOCK: find_payment executed",
		a.logger.String("identifier_hash", observability.HashForLog(identifier)),
		a.logger.Int("identifier_length", observability.LenForLog(identifier)),
		a.logger.String("provider", audit.Provider),
		a.logger.String("source", audit.Source),
		a.logger.String("status", audit.Status),
		a.logger.Int64("duration_ms", audit.DurationMS),
		a.logger.String("error_code", audit.ErrorCode))

	return nil
}

func paymentActionResult(response appprovider.PaymentLookupResponse, audit appprovider.ActionAudit) map[string]any {
	result := map[string]any{
		"status": audit.Status,
		"found":  response.Found,
		"source": response.Source,
	}
	addProviderErrorCode(result, audit)
	if response.PaymentID != "" {
		result["payment_id"] = response.PaymentID
	}
	if response.Amount != 0 {
		result["amount"] = response.Amount
	}
	if response.Currency != "" {
		result["currency"] = response.Currency
	}
	if response.Date != "" {
		result["date"] = response.Date
	}
	if response.Status != "" {
		result["payment_status"] = response.Status
	}
	if response.Purpose != "" {
		result["purpose"] = response.Purpose
	}
	if response.CreatedAt != "" {
		result["created_at"] = response.CreatedAt
	}
	return result
}
