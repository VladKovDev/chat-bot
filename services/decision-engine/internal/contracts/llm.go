package contracts

// LLMMessage format for /decide endpoint
type LLMMessage struct {
	Role string `json:"role"` // "user", "bot", "operator"
	Text string `json:"text"`
}

// DecideLLMRequest - request to LLM /decide endpoint
type DecideLLMRequest struct {
	State    string       `json:"state"`
	Summary  string       `json:"summary"`
	Messages []LLMMessage `json:"messages"`
}

// DecideLLMResponse - response from LLM /decide endpoint
type DecideLLMResponse struct {
	Intent  string   `json:"intent"`
	State   string   `json:"state"`
	Actions []string `json:"actions"`
}