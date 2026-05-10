package presenter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

const quickReplyActionSend = "send_text"

type QuickReplyConfig struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload,omitempty"`
	Order   int            `json:"order,omitempty"`
}

// ResponseConfig represents a response template from JSON
type ResponseConfig struct {
	Message      string             `json:"message"`
	Options      []string           `json:"options,omitempty"`
	QuickReplies []QuickReplyConfig `json:"quick_replies,omitempty"`
}

// Loader loads response templates from JSON file
type Presenter struct {
	configPath string
	responses  map[string]*ResponseConfig
	logger     logger.Logger
}

// NewPresenter creates a new presenter
func NewPresenter(configPath string, logs ...logger.Logger) (*Presenter, error) {
	log := logger.Noop()
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	}

	p := &Presenter{
		configPath: configPath,
		responses:  make(map[string]*ResponseConfig),
		logger:     log,
	}

	if err := p.load(); err != nil {
		return nil, err
	}

	return p, nil

}

// Present creates a response from a template key and state
func (p *Presenter) Present(responseKey string, st state.State) (response.Response, error) {
	return p.Render(RenderInput{
		ResponseKey: responseKey,
		State:       st,
	})
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

func (c *ResponseConfig) legacyOptions() []string {
	if len(c.Options) > 0 {
		return append([]string(nil), c.Options...)
	}

	if len(c.QuickReplies) == 0 {
		return nil
	}

	options := make([]string, 0, len(c.QuickReplies))
	for _, quickReply := range c.QuickReplies {
		if quickReply.Label == "" {
			continue
		}
		options = append(options, quickReply.Label)
	}

	return options
}

func (c *ResponseConfig) quickReplies() []response.QuickReply {
	if len(c.QuickReplies) > 0 {
		replies := make([]response.QuickReply, 0, len(c.QuickReplies))
		for _, quickReply := range c.QuickReplies {
			replies = append(replies, response.QuickReply{
				ID:      quickReply.ID,
				Label:   quickReply.Label,
				Action:  quickReply.Action,
				Payload: clonePayload(quickReply.Payload),
				Order:   quickReply.Order,
			})
		}
		return replies
	}

	if len(c.Options) == 0 {
		return nil
	}

	replies := make([]response.QuickReply, 0, len(c.Options))
	for index, option := range c.Options {
		replies = append(replies, response.QuickReply{
			ID:     slugifyQuickReplyID(option),
			Label:  option,
			Action: quickReplyActionSend,
			Payload: map[string]any{
				"text": option,
			},
			Order: index,
		})
	}

	return replies
}

func clonePayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}

	return cloned
}

func slugifyQuickReplyID(label string) string {
	if label == "" {
		return "quick-reply"
	}

	buf := make([]rune, 0, len(label))
	lastDash := false

	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				r = r - 'A' + 'a'
			}
			buf = append(buf, r)
			lastDash = false
			continue
		}

		if lastDash {
			continue
		}

		buf = append(buf, '-')
		lastDash = true
	}

	id := string(buf)
	for len(id) > 0 && id[0] == '-' {
		id = id[1:]
	}
	for len(id) > 0 && id[len(id)-1] == '-' {
		id = id[:len(id)-1]
	}
	if id == "" {
		return "quick-reply"
	}

	return id
}
