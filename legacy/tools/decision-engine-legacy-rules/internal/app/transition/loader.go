package transition

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
)

// Config represents the transitions configuration loaded from JSON
type Config struct {
	Transitions  []TransitionConfigJson           `json:"transitions"`
	GlobalEvents map[string]GlobalEventConfigJson `json:"global_events"`
}

// TransitionConfigJson represents a transition in JSON (with string enums)
type TransitionConfigJson struct {
	From        string   `json:"from"`
	Event       string   `json:"event"`
	To          string   `json:"to"`
	ResponseKey string   `json:"response_key"`
	Actions     []string `json:"actions,omitempty"`
}

// GlobalEventConfigJson represents a global event in JSON
type GlobalEventConfigJson struct {
	To          string   `json:"to"`
	ResponseKey string   `json:"response_key"`
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
		From:        session.Mode(j.From),
		Event:       session.Event(j.Event),
		To:          session.Mode(j.To),
		ResponseKey: j.ResponseKey,
		Actions:     j.Actions,
	}
}

// ToGlobalEventConfig converts JSON config to domain GlobalEventConfig
func (j *GlobalEventConfigJson) ToGlobalEventConfig(event string) GlobalEventConfig {
	return GlobalEventConfig{
		Event:       session.Event(event),
		To:          session.Mode(j.To),
		ResponseKey: j.ResponseKey,
		Actions:     j.Actions,
	}
}

// BuildTransitionMaps builds transition and global event maps from config
func BuildTransitionMaps(cfg *Config) (map[session.Mode]map[session.Event]*TransitionConfig, map[session.Event]*GlobalEventConfig) {
	transitions := make(map[session.Mode]map[session.Event]*TransitionConfig)
	globalEvents := make(map[session.Event]*GlobalEventConfig)

	// Build state transitions
	for _, t := range cfg.Transitions {
		transCfg := t.ToTransitionConfig()

		if transitions[transCfg.From] == nil {
			transitions[transCfg.From] = make(map[session.Event]*TransitionConfig)
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

// ExtractResponseKeys extracts all response keys from transition config
func ExtractResponseKeys(cfg *Config) []string {
	keys := make(map[string]bool)

	// Extract from regular transitions
	for _, t := range cfg.Transitions {
		trans := t.ToTransitionConfig()
		if trans.ResponseKey != "" {
			keys[trans.ResponseKey] = true
		}
	}

	// Extract from global events
	for event, global := range cfg.GlobalEvents {
		globalEv := global.ToGlobalEventConfig(event)
		if globalEv.ResponseKey != "" {
			keys[globalEv.ResponseKey] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}

	return result
}
