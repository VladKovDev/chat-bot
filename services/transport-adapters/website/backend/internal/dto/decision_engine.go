package dto

// DecisionEngineRequest represents a request to decision engine
type DecisionEngineRequest struct {
	Text      string `json:"text"`
	SessionID string `json:"session_id"`
	Channel   string `json:"channel"`
	ClientID  string `json:"client_id"`
}

// DecisionEngineResponse represents a response from decision engine
type DecisionEngineResponse struct {
	Text        string       `json:"text"`
	Options     []string     `json:"options,omitempty"`
	State       string       `json:"state"`
	ActiveTopic string       `json:"active_topic"`
	SessionID   string       `json:"session_id"`
	Channel     string       `json:"channel"`
	ClientID    string       `json:"client_id"`
	Success     bool         `json:"success"`
	Error       *PublicError `json:"error,omitempty"`
}

type SessionRequest struct {
	Channel  string `json:"channel"`
	ClientID string `json:"client_id"`
}

type SessionResponse struct {
	SessionID   string       `json:"session_id"`
	Channel     string       `json:"channel"`
	ClientID    string       `json:"client_id"`
	State       string       `json:"state"`
	ActiveTopic string       `json:"active_topic"`
	Resumed     bool         `json:"resumed"`
	Success     bool         `json:"success"`
	Error       *PublicError `json:"error,omitempty"`
}

type PublicError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}
