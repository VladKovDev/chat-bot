package presenter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type IntentCatalog struct {
	Intents []IntentDefinition `json:"intents"`
}

type IntentDefinition struct {
	Key                 string             `json:"key"`
	Category            string             `json:"category"`
	ResolutionType      string             `json:"resolution_type"`
	ResponseKey         string             `json:"response_key,omitempty"`
	FallbackResponseKey string             `json:"fallback_response_key,omitempty"`
	KnowledgeKey        string             `json:"knowledge_key,omitempty"`
	Action              string             `json:"action,omitempty"`
	ResultResponseKeys  []string           `json:"result_response_keys,omitempty"`
	Examples            []string           `json:"examples"`
	QuickReplies        []QuickReplyConfig `json:"quick_replies,omitempty"`
	E2ECoverage         []string           `json:"e2e_coverage"`
}

func LoadIntentCatalog(configPath string) (*IntentCatalog, error) {
	catalogPath, err := resolveIntentCatalogPath(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read intent catalog: %w", err)
	}

	var catalog IntentCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse intent catalog: %w", err)
	}

	return &catalog, nil
}

func resolveIntentCatalogPath(configPath string) (string, error) {
	dir := filepath.Clean(configPath)
	if absDir, err := filepath.Abs(dir); err == nil {
		dir = absDir
	}
	if dir == "." || dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			dir = cwd
		}
	}

	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "seeds", "intents.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("failed to locate seeds/intents.json from config path %q", configPath)
}
