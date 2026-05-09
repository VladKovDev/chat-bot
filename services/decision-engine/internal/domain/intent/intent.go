package intent

type Intent string

// Communicative intents - базовые коммуникативные намерения
const (
	IntentGreeting      Intent = "greeting"
	IntentGoodbye       Intent = "goodbye"
	IntentConfirmation  Intent = "confirmation"
	IntentNegation      Intent = "negation"
	IntentGratitude     Intent = "gratitude"
	IntentClarification Intent = "clarification"
	IntentUnknown       Intent = "unknown"
)

// System intents - системные намерения для управления потоком
const (
	IntentRequestOperator   Intent = "request_operator"
	IntentResetConversation Intent = "reset_conversation"
	IntentReturnToMenu      Intent = "return_to_menu"
	IntentShowContacts      Intent = "show_contacts"
)

// Main category intents - основные категории (верхнеуровневые)
const (
	IntentBooking   Intent = "booking"   // Записи и бронирование
	IntentWorkspace Intent = "workspace" // Рабочие места
	IntentPayment   Intent = "payment"   // Оплата
	IntentTechIssue Intent = "tech_issue" // Проблемы с сайтом или входом
	IntentAccount   Intent = "account"   // Аккаунт
	IntentServices  Intent = "services"  // Услуги и правила
	IntentComplaint Intent = "complaint" // Жалобы и проблемы
	IntentOther     Intent = "other"     // Другое
)

// Informational intents - информационные намерения (что хочет узнать пользователь)
const (
	IntentAskBookingInfo      Intent = "ask_booking_info"      // Информация о записи
	IntentAskBookingStatus    Intent = "ask_booking_status"    // Статус записи
	IntentAskCancellationRules Intent = "ask_cancellation_rules" // Правила отмены
	IntentAskWorkspaceInfo    Intent = "ask_workspace_info"    // Информация о рабочих местах
	IntentAskWorkspacePrices  Intent = "ask_workspace_prices"  // Цены на рабочие места
	IntentAskWorkspaceRules   Intent = "ask_workspace_rules"   // Правила аренды
	IntentAskWorkspaceStatus  Intent = "ask_workspace_status"  // Статус брони
	IntentAskPaymentStatus    Intent = "ask_payment_status"    // Статус платежа
	IntentAskPaymentProblem   Intent = "ask_payment_problem"   // Проблема с оплатой
	IntentAskSiteProblem      Intent = "ask_site_problem"      // Проблема с сайтом
	IntentAskLoginProblem     Intent = "ask_login_problem"     // Проблема с входом
	IntentAskAccountHelp      Intent = "ask_account_help"      // Помощь с аккаунтом
	IntentAskServicesInfo     Intent = "ask_services_info"     // Информация об услугах
	IntentAskPrices           Intent = "ask_prices"           // Цены
	IntentAskRules            Intent = "ask_rules"            // Правила
	IntentAskLocation         Intent = "ask_location"         // Расположение
	IntentAskFAQ              Intent = "ask_faq"              // FAQ
	IntentReportComplaint     Intent = "report_complaint"     // Жалоба
	IntentGeneralQuestion     Intent = "general_question"     // Общий вопрос
)

// Problem type intents - типы проблем
const (
	IntentPaymentNotPassed     Intent = "payment_not_passed"     // Оплата не прошла
	IntentPaymentNotActivated  Intent = "payment_not_activated"  // Деньги списались, услуга не активирована
	IntentSiteNotLoading       Intent = "site_not_loading"       // Сайт не загружается
	IntentLoginNotWorking      Intent = "login_not_working"      // Вход не работает
	IntentCodeNotReceived      Intent = "code_not_received"      // Код не приходит
	IntentBookingNotFound      Intent = "booking_not_found"      // Запись не найдена
	IntentWorkspaceUnavailable Intent = "workspace_unavailable"  // Место недоступно
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
		string(IntentReturnToMenu),
		string(IntentShowContacts),
		// Main categories
		string(IntentBooking),
		string(IntentWorkspace),
		string(IntentPayment),
		string(IntentTechIssue),
		string(IntentAccount),
		string(IntentServices),
		string(IntentComplaint),
		string(IntentOther),
		// Informational
		string(IntentAskBookingInfo),
		string(IntentAskBookingStatus),
		string(IntentAskCancellationRules),
		string(IntentAskWorkspaceInfo),
		string(IntentAskWorkspacePrices),
		string(IntentAskWorkspaceRules),
		string(IntentAskWorkspaceStatus),
		string(IntentAskPaymentStatus),
		string(IntentAskPaymentProblem),
		string(IntentAskSiteProblem),
		string(IntentAskLoginProblem),
		string(IntentAskAccountHelp),
		string(IntentAskServicesInfo),
		string(IntentAskPrices),
		string(IntentAskRules),
		string(IntentAskLocation),
		string(IntentAskFAQ),
		string(IntentReportComplaint),
		string(IntentGeneralQuestion),
		// Problem types
		string(IntentPaymentNotPassed),
		string(IntentPaymentNotActivated),
		string(IntentSiteNotLoading),
		string(IntentLoginNotWorking),
		string(IntentCodeNotReceived),
		string(IntentBookingNotFound),
		string(IntentWorkspaceUnavailable),
	}
}