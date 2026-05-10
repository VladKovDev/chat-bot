package rule_based

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadRules(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read rules file: %w", err)
	}

	var config Config

	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal rules: %w", err)
	}
	return config, nil
}
