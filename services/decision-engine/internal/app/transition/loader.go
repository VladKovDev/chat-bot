package transition

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
)

// Config represents the transitions configuration loaded from JSON
type Config struct {
	Transitions  []TransitionConfigJson `json:"transitions"`
	GlobalEvents map[string]GlobalEventConfigJson `json:"global_events"`
}

// TransitionConfigJson represents a transition in JSON (with string enums)
type TransitionConfigJson struct {
	From        string `json:"from"`
	Event       string `json:"event"`
	To          string `json:"to"`
	ResponseKey string `json:"response_key"`
	Actions     []string `json:"actions,omitempty"`
}

// GlobalEventConfigJson represents a global event in JSON
type GlobalEventConfigJson struct {
	To          string `json:"to"`
	ResponseKey string `json:"response_key"`
	Actions     []string `json:"actions,omitempty"`
}

// LoadConfig loads transition configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// ToTransitionConfig converts JSON config to domain TransitionConfig
func (j *TransitionConfigJson) ToTransitionConfig() TransitionConfig {
	return TransitionConfig{
		From:        conversation.State(j.From),
		Event:       conversation.Event(j.Event),
		To:          conversation.State(j.To),
		ResponseKey: j.ResponseKey,
		Actions:     j.Actions,
	}
}

// ToGlobalEventConfig converts JSON config to domain GlobalEventConfig
func (j *GlobalEventConfigJson) ToGlobalEventConfig(event string) GlobalEventConfig {
	return GlobalEventConfig{
		Event:       conversation.Event(event),
		To:          conversation.State(j.To),
		ResponseKey: j.ResponseKey,
		Actions:     j.Actions,
	}
}

// BuildTransitionMaps builds transition and global event maps from config
func BuildTransitionMaps(cfg *Config) (map[conversation.State]map[conversation.Event]*TransitionConfig, map[conversation.Event]*GlobalEventConfig) {
	transitions := make(map[conversation.State]map[conversation.Event]*TransitionConfig)
	globalEvents := make(map[conversation.Event]*GlobalEventConfig)

	// Build state transitions
	for _, t := range cfg.Transitions {
		transCfg := t.ToTransitionConfig()

		if transitions[transCfg.From] == nil {
			transitions[transCfg.From] = make(map[conversation.Event]*TransitionConfig)
		}

		transitions[transCfg.From][transCfg.Event] = &transCfg
	}

	// Build global events
	for event, globalCfg := range cfg.GlobalEvents {
		globalEv := globalCfg.ToGlobalEventConfig(event)
		globalEvents[globalEv.Event] = &globalEv
	}

	return transitions, globalEvents
}