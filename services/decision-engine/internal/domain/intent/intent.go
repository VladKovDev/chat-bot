package intent

type Intent string

// Communicative intents - basic conversation flow
const (
	IntentGreeting          Intent = "greeting"
	IntentGoodbye           Intent = "goodbye"
	IntentConfirmation      Intent = "confirmation"
	IntentNegation          Intent = "negation"
	IntentGratitude         Intent = "gratitude"
	IntentClarification     Intent = "clarification"
	IntentUnknown           Intent = "unknown"
)

// System intents - control flow
const (
	IntentRequestOperator   Intent = "request_operator"
	IntentResetConversation Intent = "reset_conversation"
	IntentResolved          Intent = "resolved"
	IntentNotResolved       Intent = "not_resolved"
	IntentOperatorClosed    Intent = "operator_closed"
)

// Main category intents - high-level topics
const (
	IntentBooking      Intent = "booking"      // Запись и бронирование
	IntentWorkspace    Intent = "workspace"    // Рабочие места (аренда)
	IntentPayment      Intent = "payment"      // Оплата и финансы
	IntentTechIssue    Intent = "tech_issue"   // Проблемы с сайтом/приложением
	IntentAccount      Intent = "account"      // Аккаунт и доступ
	IntentServicesInfo Intent = "services_info" // Услуги и правила
	IntentComplaint    Intent = "complaint"    // Жалобы и инциденты
	IntentOtherInput   Intent = "other_input"  // Другое / свободный ввод
)

// Action intents - universal actions across categories
const (
	IntentCreate      Intent = "create"      // Создать новую запись/бронь
	IntentView        Intent = "view"        // Просмотреть детали/список
	IntentCancel      Intent = "cancel"      // Отменить запись/бронь
	IntentReschedule  Intent = "reschedule"  // Перенести запись/бронь
	IntentRefund      Intent = "refund"      // Запросить возврат
	IntentProblem     Intent = "problem"     // Сообщить о проблеме
	IntentSearch      Intent = "search"      // Найти по номеру/контакту
	IntentUpdate      Intent = "update"      // Изменить данные
	IntentDelete      Intent = "delete"      // Удалить аккаунт
)

// Specific action intents - for common scenarios
const (
	IntentLogin            Intent = "login"             // Войти в аккаунт
	IntentForgotPassword   Intent = "forgot_password"   // Забыл пароль
	IntentRateHelp         Intent = "rate_help"         // Оценить помощь
	IntentContactSupport   Intent = "contact_support"   // Связаться с поддержкой
)

// Sub-category intents - used with main categories for context
const (
	IntentClient      Intent = "client"  // Клиент (для booking)
	IntentMaster      Intent = "master"  // Мастер (для booking)
	IntentInfo        Intent = "info"    // Информация
	IntentPrices      Intent = "prices"  // Цены
	IntentRules       Intent = "rules"   // Правила
	IntentFAQ         Intent = "faq"     // Часто задаваемые вопросы
	IntentMasters     Intent = "masters" // Мастера
	IntentLocation    Intent = "location" // Расположение
)

// Problem type intents - used with IntentProblem
const (
	IntentProblemPaymentFailed    Intent = "payment_failed"     // Оплата не прошла
	IntentProblemDoubleCharge     Intent = "double_charge"      // Двойное списание
	IntentProblemNotActivated    Intent = "not_activated"      // Не активировано
	IntentProblemNotFound        Intent = "not_found"          // Не найдено
	IntentProblemTechIssue       Intent = "technical_issue"    // Техническая проблема
	IntentProblemMasterNoAnswer  Intent = "master_no_answer"   // Мастер не отвечает
	IntentProblemWorkspaceIssue  Intent = "workspace_issue"    // Проблема с местом
)

// All returns all available intents as a slice of strings
func All() []string {
	return []string{
		// Communicative
		string(IntentGreeting),
		string(IntentGoodbye),
		string(IntentConfirmation),
		string(IntentNegation),
		string(IntentGratitude),
		string(IntentClarification),
		string(IntentUnknown),
		// System
		string(IntentRequestOperator),
		string(IntentResetConversation),
		string(IntentResolved),
		string(IntentNotResolved),
		string(IntentOperatorClosed),
		// Main categories
		string(IntentBooking),
		string(IntentWorkspace),
		string(IntentPayment),
		string(IntentTechIssue),
		string(IntentAccount),
		string(IntentServicesInfo),
		string(IntentComplaint),
		string(IntentOtherInput),
		// Actions
		string(IntentCreate),
		string(IntentView),
		string(IntentCancel),
		string(IntentReschedule),
		string(IntentRefund),
		string(IntentProblem),
		string(IntentSearch),
		string(IntentUpdate),
		string(IntentDelete),
		// Specific actions
		string(IntentLogin),
		string(IntentForgotPassword),
		string(IntentRateHelp),
		string(IntentContactSupport),
		// Sub-categories
		string(IntentClient),
		string(IntentMaster),
		string(IntentInfo),
		string(IntentPrices),
		string(IntentRules),
		string(IntentFAQ),
		string(IntentMasters),
		string(IntentLocation),
		// Problem types
		string(IntentProblemPaymentFailed),
		string(IntentProblemDoubleCharge),
		string(IntentProblemNotActivated),
		string(IntentProblemNotFound),
		string(IntentProblemTechIssue),
		string(IntentProblemMasterNoAnswer),
		string(IntentProblemWorkspaceIssue),
	}
}