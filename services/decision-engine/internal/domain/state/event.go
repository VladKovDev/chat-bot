package state

type Event string

const (
	EventUnknown          Event = "unknown"
	EventMessageReceived  Event = "message_received"
	EventCategorySelected Event = "category_selected"
	EventRequestOperator  Event = "request_operator"
	EventResolved         Event = "resolved"
	EventNotResolved      Event = "not_resolved"
	EventOperatorClosed   Event = "operator_closed"
)
