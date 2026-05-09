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
	// Basic actions - базовые действия бота
	ActionProvideInformation  string = "provide_information"   // Предоставить информацию из базы знаний
	ActionClarifyRequest      string = "clarify_request"       // Уточнить запрос пользователя
	ActionRequestIdentifier   string = "request_identifier"    // Запросить идентификатор (номер записи, телефон)
	ActionShowStatus          string = "show_status"           // Показать статус объекта
	ActionSuggestSolution     string = "suggest_solution"      // Предложить решение из базы знаний
	ActionProvideInstruction  string = "provide_instruction"   // Дать инструкцию по действию
	ActionEscalateToOperator  string = "escalate_to_operator"  // Перевести на оператора

	// Optional actions - опциональные действия
	ActionShowContactInformation string = "show_contact_information" // Показать контактную информацию
	ActionReturnToMenu           string = "return_to_menu"           // Вернуться в главное меню

	// System actions - системные действия
	ActionResetConversation string = "reset_conversation" // Сброс диалога
	ActionLogAnalytics      string = "log_analytics"      // Логирование аналитики
)

// All returns all available action keys as a slice of strings
func All() []string {
	return []string{
		// Basic
		ActionProvideInformation,
		ActionClarifyRequest,
		ActionRequestIdentifier,
		ActionShowStatus,
		ActionSuggestSolution,
		ActionProvideInstruction,
		ActionEscalateToOperator,
		// Optional
		ActionShowContactInformation,
		ActionReturnToMenu,
		// System
		ActionResetConversation,
		ActionLogAnalytics,
	}
}