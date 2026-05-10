package actions

import (
	"context"

	appprovider "github.com/VladKovDev/chat-bot/internal/app/provider"
	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindWorkspaceBooking finds workspace bookings through the configured workspace provider.
type FindWorkspaceBooking struct {
	logger   logger.Logger
	provider appprovider.WorkspaceBookingProvider
}

// NewFindWorkspaceBooking creates a new FindWorkspaceBooking action
func NewFindWorkspaceBooking(logger logger.Logger, datasets ...*appseed.Dataset) *FindWorkspaceBooking {
	var dataset *appseed.Dataset
	if len(datasets) > 0 {
		dataset = datasets[0]
	}

	return &FindWorkspaceBooking{
		logger:   logger,
		provider: appprovider.NewMockWorkspaceService(dataset),
	}
}

func (a *FindWorkspaceBooking) Execute(ctx context.Context, data action.ActionData) error {
	identifier, identifierType := providerIdentifier(data)
	response, audit, err := a.provider.LookupWorkspaceBooking(ctx, appprovider.WorkspaceLookupRequest{
		Identifier:     identifier,
		IdentifierType: identifierType,
	})
	result := workspaceBookingActionResult(response, audit)
	if err != nil {
		result = safeProviderErrorResult(audit, err)
	}

	storeProviderOutcome(data, "workspace_booking_info", result, audit)

	a.logger.Info("MOCK: find_workspace_booking executed",
		a.logger.String("identifier", identifier),
		a.logger.String("provider", audit.Provider),
		a.logger.String("source", audit.Source),
		a.logger.String("status", audit.Status),
		a.logger.Int64("duration_ms", audit.DurationMS),
		a.logger.String("error_code", audit.ErrorCode))

	return nil
}

func workspaceBookingActionResult(response appprovider.WorkspaceLookupResponse, audit appprovider.ActionAudit) map[string]any {
	result := map[string]any{
		"status": audit.Status,
		"found":  response.Found,
		"source": response.Source,
	}
	addProviderErrorCode(result, audit)
	if response.BookingNumber != "" {
		result["booking_number"] = response.BookingNumber
	}
	if response.WorkspaceType != "" {
		result["workspace_type"] = response.WorkspaceType
	}
	if response.Date != "" {
		result["date"] = response.Date
	}
	if response.Time != "" {
		result["time"] = response.Time
	}
	if response.Duration != "" {
		result["duration"] = response.Duration
	}
	if response.Status != "" {
		result["booking_status"] = response.Status
	}
	if response.DurationHours != 0 {
		result["duration_hours"] = response.DurationHours
	}
	return result
}
