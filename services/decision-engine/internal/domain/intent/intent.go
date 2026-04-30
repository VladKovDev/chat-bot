package intent

type Intent string

const (
	IntentGreeting           Intent = "greeting"
	IntentCategorySelect     Intent = "category_select"
	IntentRequestOperator    Intent = "request_operator"
	IntentResetConversation  Intent = "reset_conversation"
	IntentResolved           Intent = "resolved"
	IntentNotResolved        Intent = "not_resolved"
	IntentOperatorClosed     Intent = "operator_closed"
	IntentUnknown            Intent = "unknown"
)