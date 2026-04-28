package dto

// DecisionEngineRequest represents a request to decision engine
type DecisionEngineRequest struct {
	Text   string `json:"text"`
	ChatID int64  `json:"chat_id,omitempty"`
}

// DecisionEngineResponse represents a response from decision engine
type DecisionEngineResponse struct {
	Text    string   `json:"text"`
	Options []string `json:"options,omitempty"`
	State   string   `json:"state"`
	ChatID  int64    `json:"chat_id"`
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
}