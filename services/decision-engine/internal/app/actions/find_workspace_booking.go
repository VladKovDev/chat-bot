package actions

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindWorkspaceBooking MOCK finds workspace booking in main service DB
type FindWorkspaceBooking struct {
	logger logger.Logger
}

// NewFindWorkspaceBooking creates a new FindWorkspaceBooking action
func NewFindWorkspaceBooking(logger logger.Logger) *FindWorkspaceBooking {
	return &FindWorkspaceBooking{logger: logger}
}

// Execute MOCK generates and returns workspace booking data
func (a *FindWorkspaceBooking) Execute(ctx context.Context, data action.ActionData) error {
	// Extract identifier from context
	identifier, _ := data.Context["provided_identifier"].(string)
	if identifier == "" {
		// Try user text
		identifier = data.UserText
	}

	// Generate mock data
	mockData := a.generateMockWorkspaceBooking(identifier, mockIdentitySeed(data.Session))

	// Store result in context for processor
	data.Context["action_result"] = mockData

	// Store in session metadata for later use
	data.Session.Metadata["workspace_booking_info"] = mockData

	a.logger.Info("MOCK: find_workspace_booking executed",
		a.logger.String("identifier", identifier),
		a.logger.String("status", mockData["status"].(string)))

	return nil
}

// generateMockWorkspaceBooking MOCK generates varied workspace booking records
func (a *FindWorkspaceBooking) generateMockWorkspaceBooking(input string, identitySeed string) map[string]interface{} {
	// Simulate "not_found" for certain patterns
	if input == "INVALID" || input == "NOTFOUND" {
		return map[string]interface{}{
			"status": "not_found",
			"error":  "workspace booking not found",
		}
	}

	// Deterministic hash-based selection
	hash := fnv.New32a()
	hash.Write([]byte(fmt.Sprintf("%s:%s", identitySeed, input)))
	variant := int(hash.Sum32()) % 4

	bookings := []map[string]interface{}{
		{
			"status":         "found",
			"booking_number": "WRK-HOT-001",
			"workspace_type": "hot_seat",
			"workspace_name": "Горячее место",
			"date":           "15.05.2026",
			"time_start":     "09:00",
			"time_end":       "13:00",
			"booking_status": "confirmed",
			"price_per_hour": 200,
			"total_price":    800,
			"duration_hours": 4,
			"address":        "ул. Примерная, д. 1, офис 201",
		},
		{
			"status":         "found",
			"booking_number": "WRK-FIX-002",
			"workspace_type": "fixed_desk",
			"workspace_name": "Фиксированное место",
			"date":           "16.05.2026",
			"time_start":     "10:00",
			"time_end":       "18:00",
			"booking_status": "pending",
			"price_per_hour": 400,
			"total_price":    3200,
			"duration_hours": 8,
			"address":        "ул. Примерная, д. 1, офис 305",
			"desk_number":    "F-12",
		},
		{
			"status":         "found",
			"booking_number": "WRK-OFC1-003",
			"workspace_type": "office_1_3",
			"workspace_name": "Офис 1-3 человека",
			"date":           "17.05.2026",
			"time_start":     "09:00",
			"time_end":       "11:00",
			"booking_status": "active",
			"price_per_hour": 1500,
			"total_price":    3000,
			"duration_hours": 2,
			"address":        "ул. Примерная, д. 1, офис 401",
			"capacity":       3,
			"amenities":      []string{"Wi-Fi", "Кофе-поинт", "Проектор"},
		},
		{
			"status":         "found",
			"booking_number": "WRK-OFC4-004",
			"workspace_type": "office_4_8",
			"workspace_name": "Офис 4-8 человек",
			"date":           "14.05.2026",
			"time_start":     "10:00",
			"time_end":       "14:00",
			"booking_status": "completed",
			"price_per_hour": 3000,
			"total_price":    12000,
			"duration_hours": 4,
			"address":        "ул. Примерная, д. 1, офис 501",
			"capacity":       8,
			"amenities":      []string{"Wi-Fi", "Кофе-поинт", "Проектор", "Доска"},
		},
	}

	return bookings[variant]
}
