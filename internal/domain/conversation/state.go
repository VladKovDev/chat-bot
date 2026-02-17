package conversation

type State string

const (
	StateNew        State = "new"
	StateWaitingForCategory State = "waiting_for_category"
	StateWaitingClarification State = "waiting_clarification"
	StateSolutionOffered State = "solution_offered"
	StateEscalatedToOperator State = "escalated_to_operator"
	StateClosed State = "closed"	
)

type Event string

const (
	EventMessageReceived Event = "message_received"
	EventCategorySelected Event = "category_selected"
	EventRequestOperator Event = "request_operator"
	EventResolved Event = "confirmed_resolved"
	EventNotResolved Event = "confirmed_not_resolved"
	EventOperatorClosed Event = "operator_closed"
)