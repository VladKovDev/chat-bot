package actions

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindBooking MOCK finds appointment record in main service DB
type FindBooking struct {
	logger logger.Logger
}

// NewFindBooking creates a new FindBooking action
func NewFindBooking(logger logger.Logger) *FindBooking {
	return &FindBooking{logger: logger}
}

// Execute MOCK generates and returns booking data
func (a *FindBooking) Execute(ctx context.Context, data action.ActionData) error {
	// Extract identifier from context
	identifier, _ := data.Context["provided_identifier"].(string)
	if identifier == "" {
		// Try user text
		identifier = data.UserText
	}

	// Generate mock data
	mockData := a.generateMockBooking(identifier, mockIdentitySeed(data.Session))

	// Store result in context for processor
	data.Context["action_result"] = mockData

	// Store in session metadata for later use
	data.Session.Metadata["booking_info"] = mockData

	a.logger.Info("MOCK: find_booking executed",
		a.logger.String("identifier_hash", observability.HashForLog(identifier)),
		a.logger.Int("identifier_length", observability.LenForLog(identifier)),
		a.logger.String("status", mockData["status"].(string)))

	return nil
}

// generateMockBooking MOCK generates varied booking records
func (a *FindBooking) generateMockBooking(input string, identitySeed string) map[string]interface{} {
	// Simulate "not_found" for certain patterns
	if input == "INVALID" || input == "NOTFOUND" {
		return map[string]interface{}{
			"status": "not_found",
			"error":  "booking not found",
		}
	}

	// Deterministic hash-based selection
	hash := fnv.New32a()
	hash.Write([]byte(fmt.Sprintf("%s:%s", identitySeed, input)))
	variant := int(hash.Sum32()) % 4

	bookings := []map[string]interface{}{
		{
			"status":           "found",
			"booking_number":   "БРГ-482910",
			"service":          "Стрижка женская",
			"master":           "Анна Петрова",
			"date":             "15.05.2026",
			"time":             "14:30",
			"booking_status":   "confirmed",
			"price":            1500,
			"duration_minutes": 60,
		},
		{
			"status":           "found",
			"booking_number":   "БРГ-746281",
			"service":          "Маникюр",
			"master":           "Елена Сидорова",
			"date":             "16.05.2026",
			"time":             "10:00",
			"booking_status":   "pending",
			"price":            800,
			"duration_minutes": 45,
		},
		{
			"status":           "found",
			"booking_number":   "БРГ-192837",
			"service":          "Массаж",
			"master":           "Иван Иванов",
			"date":             "10.05.2026",
			"time":             "16:00",
			"booking_status":   "completed",
			"price":            2000,
			"duration_minutes": 90,
		},
		{
			"status":              "found",
			"booking_number":      "БРГ-564738",
			"service":             "Окрашивание",
			"master":              "Мария Новикова",
			"date":                "12.05.2026",
			"time":                "12:00",
			"booking_status":      "cancelled",
			"price":               3000,
			"duration_minutes":    120,
			"cancellation_reason": "client_request",
		},
	}

	return bookings[variant]
}
