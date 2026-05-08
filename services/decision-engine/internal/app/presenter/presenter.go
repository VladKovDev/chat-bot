package presenter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

// ResponseConfig represents a response template from JSON
type ResponseConfig struct {
	Message string   `json:"message"`
	Options []string `json:"options,omitempty"`
}

// Loader loads response templates from JSON file
type Presenter struct {
	configPath string
	responses      map[string]*ResponseConfig
}

// NewPresenter creates a new presenter
func NewPresenter(configPath string) (*Presenter, error) {
	p := &Presenter{
		configPath: configPath,
		responses:      make(map[string]*ResponseConfig),
	}

	if err := p.load(); err != nil {
		return nil, err
	}

	return p, nil

}

// Present creates a response from a template key and state
func (p *Presenter) Present(responseKey string, st state.State) (response.Response, error) {
	cfg, err := p.GetResponse(responseKey)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to load response config: %w", err)
	}

	return response.Response{
		Text:    cfg.Message,
		Options: cfg.Options,
		State:   st,
	}, nil
}

// load loads all response templates from JSON file
func (p *Presenter) load() error {
	data, err := os.ReadFile(p.configPath + "/responses.json")
	if err != nil {
		return fmt.Errorf("failed to read responses file: %w", err)
	}

	var responses map[string]*ResponseConfig
	if err := json.Unmarshal(data, &responses); err != nil {
		return fmt.Errorf("failed to parse responses: %w", err)
	}

	p.responses = responses
	return nil
}

// Load returns a response config by key
func (p *Presenter) GetResponse(key string) (*ResponseConfig, error) {
	if response, ok := p.responses[key]; ok {
		return response, nil
	}

	return nil, ErrKeyNotFound
}

// GetAll returns all response configs
func (p *Presenter) GetAll() map[string]*ResponseConfig {
	return p.responses
}

// GetAllKeys returns all loaded response keys
func (p *Presenter) GetAllKeys() []string {
	keys := make([]string, 0, len(p.responses))

	for key := range p.responses {
		keys = append(keys, key)
	}

	return keys
}

