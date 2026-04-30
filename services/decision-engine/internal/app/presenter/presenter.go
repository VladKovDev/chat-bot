package presenter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
)

// ResponseConfig represents a response template from JSON
type ResponseConfig struct {
	Message string   `json:"message"`
	Options []string `json:"options,omitempty"`
}

// Loader loads response templates from JSON file
type Loader struct {
	configPath string
	cache      map[string]*ResponseConfig
}

// NewLoader creates a new response loader
func NewLoader(configPath string) (*Loader, error) {
	l := &Loader{
		configPath: configPath,
		cache:      make(map[string]*ResponseConfig),
	}

	if err := l.load(); err != nil {
		return nil, err
	}

	return l, nil
}

// load loads all response templates from JSON file
func (l *Loader) load() error {
	data, err := os.ReadFile(l.configPath + "/responses.json")
	if err != nil {
		return fmt.Errorf("failed to read responses file: %w", err)
	}

	var responses map[string]*ResponseConfig
	if err := json.Unmarshal(data, &responses); err != nil {
		return fmt.Errorf("failed to parse responses: %w", err)
	}

	l.cache = responses
	return nil
}

// Load returns a response config by key
func (l *Loader) Load(key string) (*ResponseConfig, error) {
	if cfg, ok := l.cache[key]; ok {
		return cfg, nil
	}

	// Return default response if key not found
	return &ResponseConfig{
		Message: "Извините, произошла ошибка. Попробуйте позже.",
		Options: []string{},
	}, nil
}

// Presenter formats responses using loaded templates
type Presenter struct {
	loader *Loader
}

// NewPresenter creates a new presenter
func NewPresenter(loader *Loader) *Presenter {
	return &Presenter{
		loader: loader,
	}
}

// Present creates a response from a template key and state
func (p *Presenter) Present(responseKey string, state conversation.State) (response.Response, error) {
	cfg, err := p.loader.Load(responseKey)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to load response config: %w", err)
	}

	return response.Response{
		Text:    cfg.Message,
		Options: cfg.Options,
		State:   state,
	}, nil
}