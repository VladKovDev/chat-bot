package dto

type DecisionEngineRequest struct {
	Text       string      `json:"text,omitempty"`
	SessionID  string      `json:"session_id"`
	EventID    string      `json:"event_id"`
	Type       string      `json:"type"`
	Channel    string      `json:"channel"`
	ClientID   string      `json:"client_id"`
	QuickReply *QuickReply `json:"quick_reply,omitempty"`
}

type QuickReply struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload,omitempty"`
	Order   int            `json:"order,omitempty"`
}

type Handoff struct {
	HandoffID  string  `json:"handoff_id"`
	SessionID  string  `json:"session_id"`
	Status     string  `json:"status"`
	Reason     string  `json:"reason,omitempty"`
	OperatorID *string `json:"operator_id,omitempty"`
}

type DecisionEngineResponse struct {
	SessionID     string       `json:"session_id"`
	UserMessageID string       `json:"user_message_id"`
	BotMessageID  string       `json:"bot_message_id"`
	Mode          string       `json:"mode"`
	ActiveTopic   *string      `json:"active_topic"`
	Text          string       `json:"text"`
	QuickReplies  []QuickReply `json:"quick_replies,omitempty"`
	Handoff       *Handoff     `json:"handoff"`
	CorrelationID string       `json:"correlation_id"`
	Timestamp     string       `json:"timestamp"`
}

type SessionRequest struct {
	Channel  string `json:"channel"`
	ClientID string `json:"client_id"`
}

type SessionResponse struct {
	SessionID   string  `json:"session_id"`
	UserID      string  `json:"user_id"`
	Mode        string  `json:"mode"`
	ActiveTopic *string `json:"active_topic"`
	Resumed     bool    `json:"resumed"`
}

type OperatorQueueActionResponse struct {
	Handoff Handoff `json:"handoff"`
}

type SessionMessagesResponse struct {
	Items []SessionMessageRecord `json:"items"`
}

type SessionMessageRecord struct {
	MessageID  string  `json:"message_id"`
	SessionID  string  `json:"session_id"`
	SenderType string  `json:"sender_type"`
	Text       string  `json:"text"`
	Intent     *string `json:"intent,omitempty"`
	Timestamp  string  `json:"timestamp"`
}

type OperatorQueueResponse struct {
	Items []OperatorQueueItem `json:"items"`
}

type OperatorQueueItem struct {
	HandoffID       string                  `json:"handoff_id"`
	SessionID       string                  `json:"session_id"`
	Status          string                  `json:"status"`
	Reason          string                  `json:"reason"`
	OperatorID      *string                 `json:"operator_id,omitempty"`
	ActiveTopic     *string                 `json:"active_topic"`
	LastIntent      *string                 `json:"last_intent"`
	Confidence      *float64                `json:"confidence,omitempty"`
	FallbackCount   int                     `json:"fallback_count"`
	ActionSummaries []OperatorActionSummary `json:"action_summaries"`
	CreatedAt       string                  `json:"created_at"`
	Preview         string                  `json:"preview"`
}

type OperatorActionSummary struct {
	ActionType string `json:"action_type"`
	Status     string `json:"status"`
	Summary    string `json:"summary,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type OperatorQueueActionRequest struct {
	OperatorID string `json:"operator_id"`
}

type OperatorMessageRequest struct {
	OperatorID string `json:"operator_id"`
	Text       string `json:"text"`
}

type OperatorMessageResponse struct {
	SessionID     string `json:"session_id"`
	MessageID     string `json:"message_id"`
	OperatorID    string `json:"operator_id"`
	Text          string `json:"text"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"`
}

type PublicError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type ErrorEnvelope struct {
	Error PublicError `json:"error"`
}
