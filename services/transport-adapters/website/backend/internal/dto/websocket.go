package dto

type ClientEvent struct {
	Type          string      `json:"type"`
	SessionID     string      `json:"session_id,omitempty"`
	EventID       string      `json:"event_id,omitempty"`
	CorrelationID string      `json:"correlation_id,omitempty"`
	Timestamp     string      `json:"timestamp,omitempty"`
	Text          string      `json:"text,omitempty"`
	ClientID      string      `json:"client_id,omitempty"`
	QuickReply    *QuickReply `json:"quick_reply,omitempty"`
}

type EventEnvelope struct {
	Type          string `json:"type"`
	SessionID     string `json:"session_id,omitempty"`
	EventID       string `json:"event_id"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"`
}

type SessionStartedEvent struct {
	EventEnvelope
	Mode        string  `json:"mode"`
	ActiveTopic *string `json:"active_topic"`
	Resumed     bool    `json:"resumed"`
}

type MessageBotEvent struct {
	EventEnvelope
	MessageID    string       `json:"message_id"`
	Text         string       `json:"text"`
	QuickReplies []QuickReply `json:"quick_replies,omitempty"`
	Mode         string       `json:"mode"`
	ActiveTopic  *string      `json:"active_topic"`
}

type MessageOperatorEvent struct {
	EventEnvelope
	MessageID  string  `json:"message_id"`
	OperatorID string  `json:"operator_id,omitempty"`
	Text       string  `json:"text"`
	Mode       string  `json:"mode,omitempty"`
	ActiveTopic *string `json:"active_topic,omitempty"`
}

type HandoffEvent struct {
	EventEnvelope
	Handoff Handoff `json:"handoff"`
}

type ErrorEvent struct {
	EventEnvelope
	Error PublicError `json:"error"`
}

const (
	EventSessionStart      = "session.start"
	EventMessageUser       = "message.user"
	EventQuickReplySelected = "quick_reply.selected"
	EventOperatorClose     = "operator.close"

	EventSessionStarted   = "session.started"
	EventMessageBot       = "message.bot"
	EventMessageOperator  = "message.operator"
	EventHandoffQueued    = "handoff.queued"
	EventHandoffAccepted  = "handoff.accepted"
	EventHandoffClosed    = "handoff.closed"
	EventError            = "error"
)
