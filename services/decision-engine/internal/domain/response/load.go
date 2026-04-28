package response

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type jsonResponse map[string]struct {
	Message string   `json:"message"`
	Options []string `json:"options"`
}

type ResponseLoader struct {
	responses jsonResponse
	mu        sync.RWMutex
}

// NewResponseLoader creates a new ResponseLoader and loads responses from file
func NewResponseLoader(configPath string) (*ResponseLoader, error) {
	rl := &ResponseLoader{}

	if err := rl.Load(configPath); err != nil {
		return nil, err
	}

	return rl, nil
}

// Load reads and parses the responses.json file
func (rl *ResponseLoader) Load(configPath string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	jsonPath := filepath.Join(configPath, "responses.json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	var resp jsonResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	rl.responses = resp

	return nil
}

// GetResponse retrieves message and options by key
func (rl *ResponseLoader) GetResponse(key string) (message string, options []string, ok bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	resp, exists := rl.responses[key]
	if !exists {
		return "", nil, false
	}

	return resp.Message, resp.Options, true
}

// GetMessage retrieves only the message by key
func (rl *ResponseLoader) GetMessage(key string) (string, bool) {
	message, _, ok := rl.GetResponse(key)
	return message, ok
}

// GetOptions retrieves only the options by key
func (rl *ResponseLoader) GetOptions(key string) ([]string, bool) {
	_, options, ok := rl.GetResponse(key)
	return options, ok
}

// HasKey checks if a key exists in the responses
func (rl *ResponseLoader) HasKey(key string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	_, exists := rl.responses[key]
	return exists
}