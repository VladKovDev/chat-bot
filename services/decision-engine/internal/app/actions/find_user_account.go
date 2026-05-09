package actions

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// FindUserAccount MOCK finds user account in main service DB
type FindUserAccount struct {
	logger logger.Logger
}

// NewFindUserAccount creates a new FindUserAccount action
func NewFindUserAccount(logger logger.Logger) *FindUserAccount {
	return &FindUserAccount{logger: logger}
}

// Execute MOCK generates and returns user account data
func (a *FindUserAccount) Execute(ctx context.Context, data action.ActionData) error {
	// Extract identifier from context
	identifier, _ := data.Context["provided_identifier"].(string)
	if identifier == "" {
		// Try user text
		identifier = data.UserText
	}

	// Generate mock data
	mockData := a.generateMockUserAccount(identifier, data.Session.ChatID)

	// Store result in context for processor
	data.Context["action_result"] = mockData

	// Store in session metadata for later use
	data.Session.Metadata["user_account_info"] = mockData

	a.logger.Info("MOCK: find_user_account executed",
		a.logger.String("identifier", identifier),
		a.logger.String("status", mockData["status"].(string)))

	return nil
}

// generateMockUserAccount MOCK generates varied user account records
func (a *FindUserAccount) generateMockUserAccount(input string, chatID int64) map[string]interface{} {
	// Special patterns for testing
	if input == "usr-NOTFOUND" || input == "INVALID" || input == "NOTFOUND" {
		return map[string]interface{}{
			"status": "not_found",
			"error":  "user account not found",
		}
	}

	if input == "usr-ZERO" {
		return map[string]interface{}{
			"status":      "found",
			"user_id":     "usr-000001",
			"phone":       "+7 (999) 123-45-67",
			"email":       "user1@example.com",
			"name":        "Иван Иванов",
			"balance":     0,
			"bonus":       0,
			"currency":    "RUB",
			"account_status": "active",
			"created_at":  "01.01.2025",
		}
	}

	// Deterministic hash-based selection
	hash := fnv.New32a()
	hash.Write([]byte(fmt.Sprintf("%d:%s", chatID, input)))
	variant := int(hash.Sum32()) % 4

	accounts := []map[string]interface{}{
		{
			"status":         "found",
			"user_id":        "usr-000001",
			"phone":          "+7 (999) 123-45-67",
			"email":          "user1@example.com",
			"name":           "Иван Иванов",
			"balance":        0,
			"bonus":          0,
			"currency":       "RUB",
			"account_status": "active",
			"created_at":     "01.01.2025",
		},
		{
			"status":         "found",
			"user_id":        "usr-000002",
			"phone":          "+7 (916) 234-56-78",
			"email":          "user2@example.com",
			"name":           "Петр Петров",
			"balance":        150,
			"bonus":          50,
			"currency":       "RUB",
			"account_status": "active",
			"created_at":     "15.02.2025",
		},
		{
			"status":         "found",
			"user_id":        "usr-000003",
			"phone":          "+7 (926) 345-67-89",
			"email":          "user3@example.com",
			"name":           "Сидор Сидоров",
			"balance":        1200,
			"bonus":          200,
			"currency":       "RUB",
			"account_status": "active",
			"created_at":     "01.03.2025",
		},
		{
			"status":         "found",
			"user_id":        "usr-000004",
			"phone":          "+7 (936) 456-78-90",
			"email":          "user4@example.com",
			"name":           "Алексей Алексеев",
			"balance":        3500,
			"bonus":          500,
			"currency":       "RUB",
			"account_status": "vip",
			"created_at":     "10.01.2025",
		},
	}

	return accounts[variant]
}
