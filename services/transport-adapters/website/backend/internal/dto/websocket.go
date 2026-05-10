package dto

// WSMessage represents a WebSocket message from client
type WSMessage struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	ClientID string `json:"client_id,omitempty"`
}

// WSResponse represents a WebSocket response to client
type WSResponse struct {
	Type        string   `json:"type"`
	Text        string   `json:"text"`
	Options     []string `json:"options,omitempty"`
	State       string   `json:"state,omitempty"`
	ActiveTopic string   `json:"active_topic,omitempty"`
	SessionID   string   `json:"session_id,omitempty"`
	Resumed     bool     `json:"resumed,omitempty"`
	Timestamp   string   `json:"timestamp,omitempty"`
}

// WSError represents a WebSocket error response
type WSError struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Code string `json:"code,omitempty"`
}

// Message types
const (
	MessageTypeUser    = "message"
	MessageTypeBot     = "response"
	MessageTypeError   = "error"
	MessageTypeSession = "session"
)
