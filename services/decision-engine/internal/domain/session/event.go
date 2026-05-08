package session

type Event string

const (
	EventUnknown           Event = "unknown"
	EventGreeting          Event = "greeting"
	EventMessageReceived   Event = "message_received"
	EventCategorySelected  Event = "category_selected"
	EventRequestOperator   Event = "request_operator"
	EventResetConversation Event = "reset_conversation"
	EventResolved          Event = "resolved"
	EventNotResolved       Event = "not_resolved"
	EventOperatorClosed    Event = "operator_closed"
	EventConfirmation      Event = "confirmation"
	EventNegation          Event = "negation"
	EventGratitude         Event = "gratitude"
	EventClarification     Event = "clarification"
)
