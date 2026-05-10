package actions

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/observability"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindPayment MOCK finds payment record in main service DB
type FindPayment struct {
	logger logger.Logger
}

// NewFindPayment creates a new FindPayment action
func NewFindPayment(logger logger.Logger) *FindPayment {
	return &FindPayment{logger: logger}
}

// Execute MOCK generates and returns payment data
func (a *FindPayment) Execute(ctx context.Context, data action.ActionData) error {
	// Extract identifier from context
	identifier, _ := data.Context["provided_identifier"].(string)
	if identifier == "" {
		// Try user text
		identifier = data.UserText
	}

	// Generate mock data
	mockData := a.generateMockPayment(identifier, data.Session.ChatID)

	// Store result in context for processor
	data.Context["action_result"] = mockData

	// Store in session metadata for later use
	data.Session.Metadata["payment_info"] = mockData

	a.logger.Info("MOCK: find_payment executed",
		a.logger.String("identifier_hash", observability.HashForLog(identifier)),
		a.logger.Int("identifier_length", observability.LenForLog(identifier)),
		a.logger.String("status", mockData["status"].(string)))

	return nil
}

// generateMockPayment MOCK generates varied payment records
func (a *FindPayment) generateMockPayment(input string, chatID int64) map[string]interface{} {
	// Special patterns for testing
	if input == "PAY-NOTFOUND" || input == "INVALID" || input == "NOTFOUND" {
		return map[string]interface{}{
			"status": "not_found",
			"error":  "payment not found",
		}
	}

	if input == "PAY-FAILED" {
		return map[string]interface{}{
			"status":         "found",
			"payment_id":     "PAY-999999",
			"amount":         1500,
			"currency":       "RUB",
			"payment_status": "failed",
			"purpose":        "Офис 1-3 чел (1 час)",
			"created_at":     "13.05.2026 10:30",
			"error_reason":   "insufficient_funds",
		}
	}

	// Deterministic hash-based selection (5 variants)
	hash := fnv.New32a()
	hash.Write([]byte(fmt.Sprintf("%d:%s", chatID, input)))
	variant := int(hash.Sum32()) % 5

	payments := []map[string]interface{}{
		{
			"status":         "found",
			"payment_id":     "PAY-000001",
			"amount":         0,
			"currency":       "RUB",
			"payment_status": "pending",
			"purpose":        "Бронирование рабочего места",
			"created_at":     "15.05.2026 09:00",
		},
		{
			"status":         "found",
			"payment_id":     "PAY-123456",
			"amount":         200,
			"currency":       "RUB",
			"payment_status": "completed",
			"purpose":        "Горячее место (1 час)",
			"created_at":     "14.05.2026 10:15",
			"completed_at":   "14.05.2026 10:16",
			"payment_method": "card",
		},
		{
			"status":         "found",
			"payment_id":     "PAY-789012",
			"amount":         1500,
			"currency":       "RUB",
			"payment_status": "failed",
			"purpose":        "Офис 1-3 чел (1 час)",
			"created_at":     "13.05.2026 14:30",
			"error_reason":   "transaction_declined",
		},
		{
			"status":         "found",
			"payment_id":     "PAY-456789",
			"amount":         5000,
			"currency":       "RUB",
			"payment_status": "refunded",
			"purpose":        "Офис 4-8 чел (2 часа)",
			"created_at":     "12.05.2026 11:00",
			"completed_at":   "12.05.2026 11:02",
			"refunded_at":    "13.05.2026 09:30",
			"refund_reason":  "client_request",
			"payment_method": "card",
		},
		{
			"status":         "found",
			"payment_id":     "PAY-345678",
			"amount":         1000,
			"currency":       "RUB",
			"payment_status": "pending",
			"purpose":        "Стрижка мужская",
			"created_at":     "15.05.2026 08:00",
			"payment_method": "cash",
		},
	}

	return payments[variant]
}
