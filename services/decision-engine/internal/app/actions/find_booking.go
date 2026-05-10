package actions

import (
	"context"

	appprovider "github.com/VladKovDev/chat-bot/internal/app/provider"
	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindBooking finds appointment records through the configured booking provider.
type FindBooking struct {
	logger   logger.Logger
	provider appprovider.BookingProvider
}

// NewFindBooking creates a new FindBooking action
func NewFindBooking(logger logger.Logger, datasets ...*appseed.Dataset) *FindBooking {
	var dataset *appseed.Dataset
	if len(datasets) > 0 {
		dataset = datasets[0]
	}

	return &FindBooking{
		logger:   logger,
		provider: appprovider.NewMockBookingService(dataset),
	}
}

func (a *FindBooking) Execute(ctx context.Context, data action.ActionData) error {
	identifier, identifierType := providerIdentifier(data)
	response, audit, err := a.provider.LookupBooking(ctx, appprovider.BookingLookupRequest{
		Identifier:     identifier,
		IdentifierType: identifierType,
	})
	result := bookingActionResult(response, audit)
	if err != nil {
		result = safeProviderErrorResult(audit, err)
	}

	storeProviderOutcome(data, "booking_info", result, audit)

	a.logger.Info("MOCK: find_booking executed",
		a.logger.String("identifier_hash", observability.HashForLog(identifier)),
		a.logger.Int("identifier_length", observability.LenForLog(identifier)),
		a.logger.String("provider", audit.Provider),
		a.logger.String("source", audit.Source),
		a.logger.String("status", audit.Status),
		a.logger.Int64("duration_ms", audit.DurationMS),
		a.logger.String("error_code", audit.ErrorCode))

	return nil
}

func bookingActionResult(response appprovider.BookingLookupResponse, audit appprovider.ActionAudit) map[string]any {
	result := map[string]any{
		"status": audit.Status,
		"found":  response.Found,
		"source": response.Source,
	}
	addProviderErrorCode(result, audit)
	if response.BookingNumber != "" {
		result["booking_number"] = response.BookingNumber
	}
	if response.Service != "" {
		result["service"] = response.Service
	}
	if response.Master != "" {
		result["master"] = response.Master
	}
	if response.Date != "" {
		result["date"] = response.Date
	}
	if response.Time != "" {
		result["time"] = response.Time
	}
	if response.Status != "" {
		result["booking_status"] = response.Status
	}
	if response.Price != 0 {
		result["price"] = response.Price
	}
	return result
}
