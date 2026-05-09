package action

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
)

// Action represents a business operation that can be executed during a transition
type Action interface {
	Execute(ctx context.Context, data ActionData) error
}

// ActionData contains information needed to execute an action
type ActionData struct {
	Session   *session.Session
	UserText  string
	Context   map[string]interface{}
}

// Action keys - string identifiers for actions
// Бот НЕ выполняет бизнес-операции (создание, отмена, перенос записей, возвраты)
// Бот ТОЛЬКО предоставляет информацию, уточняет запросы, показывает статус, предлагает решения

const (
	// Business actions - поиск информации в основной БД (read-only)
	ActionFindBooking           string = "find_booking"            // Найти запись на услугу (салон)
	ActionFindWorkspaceBooking  string = "find_workspace_booking"  // Найти бронь рабочего места (коворкинг)
	ActionFindPayment           string = "find_payment"            // Найти платеж
	ActionFindUserAccount       string = "find_user_account"       // Найти пользовательский аккаунт

	// Utility actions - вспомогательные действия
	ActionValidateIdentifier    string = "validate_identifier"     // Валидация формата идентификатора
	ActionEscalateToOperator    string = "escalate_to_operator"    // Перевести на оператора
	ActionResetConversation     string = "reset_conversation"      // Сброс диалога
)

// All returns all available action keys as a slice of strings
func All() []string {
	return []string{
		// Business actions
		ActionFindBooking,
		ActionFindWorkspaceBooking,
		ActionFindPayment,
		ActionFindUserAccount,
		// Utility actions
		ActionValidateIdentifier,
		ActionEscalateToOperator,
		ActionResetConversation,
	}
}