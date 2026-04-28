package dto

// WSMessage represents a WebSocket message from client
type WSMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// WSResponse represents a WebSocket response to client
type WSResponse struct {
	Type      string   `json:"type"`
	Text      string   `json:"text"`
	Options   []string `json:"options,omitempty"`
	State     string   `json:"state,omitempty"`
	Timestamp string   `json:"timestamp,omitempty"`
}

// WSError represents a WebSocket error response
type WSError struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Code  string `json:"code,omitempty"`
}

// Message types
const (
	MessageTypeUser   = "message"
	MessageTypeBot    = "response"
	MessageTypeError  = "error"
)