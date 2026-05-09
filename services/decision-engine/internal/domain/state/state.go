package state

type State string

// General states - базовые состояния диалога
const (
	StateNew                  State = "new"
	StateWaitingForCategory   State = "waiting_for_category"
	StateWaitingClarification State = "waiting_clarification"
	StateWaitingForIdentifier State = "waiting_for_identifier"
	StateEscalatedToOperator  State = "escalated_to_operator"
	StateClosed               State = "closed"
)

// Category states - состояния для основных категорий
const (
	StateBooking       State = "booking"        // Записи и бронирование
	StateWorkspace     State = "workspace"      // Рабочие места
	StatePayment       State = "payment"        // Оплата
	StateTechIssue     State = "tech_issue"     // Проблемы с сайтом/входом
	StateAccount       State = "account"        // Аккаунт
	StateServices      State = "services"       // Услуги и правила
	StateComplaint     State = "complaint"      // Жалобы и проблемы
	StateOther         State = "other"          // Другое
)

// Information states - состояния для предоставления информации
const (
	StateProvidingInfo    State = "providing_info"     // Предоставление информации
	StateShowingStatus    State = "showing_status"     // Показ статуса
	StateProvidingInstruction State = "providing_instruction" // Предоставление инструкции
	StateSuggestingSolution State = "suggesting_solution" // Предложение решения
)

// Contact information state
const (
	StateShowContactInfo State = "show_contact_info" // Показ контактной информации
)

// All returns all available states as a slice of strings
func All() []string {
	return []string{
		// General
		string(StateNew),
		string(StateWaitingForCategory),
		string(StateWaitingClarification),
		string(StateWaitingForIdentifier),
		string(StateEscalatedToOperator),
		string(StateClosed),
		// Categories
		string(StateBooking),
		string(StateWorkspace),
		string(StatePayment),
		string(StateTechIssue),
		string(StateAccount),
		string(StateServices),
		string(StateComplaint),
		string(StateOther),
		// Information
		string(StateProvidingInfo),
		string(StateShowingStatus),
		string(StateProvidingInstruction),
		string(StateSuggestingSolution),
		// Contact
		string(StateShowContactInfo),
	}
}
