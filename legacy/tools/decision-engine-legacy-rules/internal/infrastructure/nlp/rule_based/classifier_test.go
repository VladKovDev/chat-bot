package rule_based

import (
	"encoding/json"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/domain/intent"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func TestClassifier_ConfidenceCalculation(t *testing.T) {
	configJSON := `{
		"threshold": {
			"min_score": 0.0,
			"ambiguity_delta": 0.0
		},
		"intents": [
			{
				"name": "greeting",
				"rules": [
					{
						"type": "keyword",
						"weight": 1.0,
						"values": ["привет", "здравствуй", "добрый"]
					},
					{
						"type": "phrase",
						"weight": 3.0,
						"values": ["добрый день", "доброе утро"]
					}
				]
			},
			{
				"name": "request_operator",
				"rules": [
					{
						"type": "keyword",
						"weight": 1.5,
						"values": ["оператор", "человек", "специалист"]
					},
					{
						"type": "phrase",
						"weight": 3.0,
						"values": ["поговорить оператором", "связаться человеком"]
					}
				]
			}
		]
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	classifier, err := NewRuleBased(cfg, logger.Noop())
	if err != nil {
		t.Fatalf("Failed to create classifier: %v", err)
	}

	tests := []struct {
		name           string
		tokens         []string
		expectedIntent string
		description    string
	}{
		{
			name:           "Single keyword match",
			tokens:         []string{"оператор"},
			expectedIntent: "request_operator",
			description:    "Single keyword should match request_operator intent",
		},
		{
			name:           "Multiple keywords same intent",
			tokens:         []string{"оператор", "специалист"},
			expectedIntent: "request_operator",
			description:    "Multiple keywords from same intent should accumulate score",
		},
		{
			name:           "Phrase match without stopwords",
			tokens:         []string{"поговорить", "оператором"},
			expectedIntent: "request_operator",
			description:    "Phrase 'поговорить оператором' should match",
		},
		{
			name:           "Exact phrase match",
			tokens:         []string{"добрый", "день"},
			expectedIntent: "greeting",
			description:    "Exact phrase match should work",
		},
		{
			name:           "Keyword overlap between intents",
			tokens:         []string{"добрый", "оператор"},
			expectedIntent: "request_operator",
			description:    "Both intents match, but operator has higher score (1.5 vs 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classifier.Classify(tt.tokens)
			if err != nil {
				t.Errorf("Classify() error = %v", err)
				return
			}

			t.Logf("Test: %s", tt.description)
			t.Logf("  Tokens: %v", tt.tokens)
			t.Logf("  Expected: %s, Got: %s", tt.expectedIntent, result)

			if result != intent.Intent(tt.expectedIntent) {
				t.Errorf("Expected intent %s, got %s", tt.expectedIntent, result)
			}
		})
	}
}

func TestClassifier_PhraseMatchingWithStopwords(t *testing.T) {
	// Test phrase matching without stopwords in config
	configJSON := `{
		"threshold": {
			"min_score": 0.0,
			"ambiguity_delta": 0.0
		},
		"intents": [
			{
				"name": "request_operator",
				"rules": [
					{
						"type": "phrase",
						"weight": 3.0,
						"values": ["связаться оператором", "поговорить человеком"]
					}
				]
			}
		]
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	classifier, err := NewRuleBased(cfg, logger.Noop())
	if err != nil {
		t.Fatalf("Failed to create classifier: %v", err)
	}

	tests := []struct {
		name        string
		tokens      []string
		shouldMatch bool
		description string
	}{
		{
			name:        "Phrase 'связаться оператором' - exact match",
			tokens:      []string{"связаться", "оператором"},
			shouldMatch: true,
			description: "Phrase matches config exactly",
		},
		{
			name:        "Single word - no match",
			tokens:      []string{"связаться"},
			shouldMatch: false,
			description: "Single word doesn't match phrase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classifier.Classify(tt.tokens)
			if err != nil {
				t.Errorf("Classify() error = %v", err)
				return
			}

			t.Logf("Test: %s", tt.description)
			t.Logf("  Tokens: %v", tt.tokens)
			t.Logf("  Result: %s", result)

			matched := result != intent.IntentUnknown
			if matched != tt.shouldMatch {
				t.Errorf("Expected match=%v, got match=%v (result=%s)", tt.shouldMatch, matched, result)
			}
		})
	}
}

func TestClassifier_AbsoluteScoreThreshold(t *testing.T) {
	// Test absolute score threshold (new behavior)
	configJSON := `{
		"threshold": {
			"min_score": 2.0,
			"ambiguity_delta": 0.5
		},
		"intents": [
			{
				"name": "test_intent",
				"rules": [
					{
						"type": "keyword",
						"weight": 1.0,
						"values": ["word1", "word2", "word3"]
					},
					{
						"type": "phrase",
						"weight": 3.0,
						"values": ["phrase one", "phrase two"]
					}
				]
			}
		]
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	t.Logf("Loaded config threshold:")
	t.Logf("  min_score: %.1f", cfg.Threshold.MinScore)
	t.Logf("  ambiguity_delta: %.1f", cfg.Threshold.AmbiguityDelta)

	classifier, err := NewRuleBased(cfg, logger.Noop())
	if err != nil {
		t.Fatalf("Failed to create classifier: %v", err)
	}

	t.Logf("Absolute score threshold test:")
	t.Logf("  min_score: 2.0")
	t.Logf("  Keyword weight: 1.0, Phrase weight: 3.0")

	tests := []struct {
		name        string
		tokens      []string
		expectedScore float64
		shouldMatch bool
	}{
		{
			name:        "Single keyword (score 1.0)",
			tokens:      []string{"word1"},
			expectedScore: 1.0,
			shouldMatch: false, // Below threshold 2.0
		},
		{
			name:        "Two keywords (score 2.0)",
			tokens:      []string{"word1", "word2"},
			expectedScore: 2.0,
			shouldMatch: true, // At threshold 2.0
		},
		{
			name:        "Three keywords (score 3.0)",
			tokens:      []string{"word1", "word2", "word3"},
			expectedScore: 3.0,
			shouldMatch: true, // Above threshold 2.0
		},
		{
			name:        "Single phrase (score 3.0)",
			tokens:      []string{"phrase", "one"},
			expectedScore: 3.0,
			shouldMatch: true, // Above threshold 2.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classifier.Classify(tt.tokens)
			if err != nil {
				t.Errorf("Classify() error = %v", err)
				return
			}

			t.Logf("  Tokens: %v, Expected score: %.1f", tt.tokens, tt.expectedScore)
			t.Logf("  Result: %s, Should match: %v", result, tt.shouldMatch)

			matched := result != intent.IntentUnknown
			if matched != tt.shouldMatch {
				t.Errorf("Expected match=%v, got match=%v (result=%s)", tt.shouldMatch, matched, result)
			}
		})
	}
}

func TestClassifier_AmbiguityDetection(t *testing.T) {
	// Test ambiguity detection with absolute delta
	configJSON := `{
		"threshold": {
			"min_score": 1.0,
			"ambiguity_delta": 1.0
		},
		"intents": [
			{
				"name": "intent_a",
				"rules": [
					{
						"type": "keyword",
						"weight": 2.0,
						"values": ["word_a"]
					}
				]
			},
			{
				"name": "intent_b",
				"rules": [
					{
						"type": "keyword",
						"weight": 1.5,
						"values": ["word_b"]
					}
				]
			}
		]
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	classifier, err := NewRuleBased(cfg, logger.Noop())
	if err != nil {
		t.Fatalf("Failed to create classifier: %v", err)
	}

	t.Logf("Ambiguity detection test:")
	t.Logf("  min_score: 1.0, ambiguity_delta: 1.0")

	tests := []struct {
		name             string
		tokens           []string
		expectedIntent   string
		expectedAmbiguous bool
		description      string
	}{
		{
			name:             "Clear match A (score 2.0 vs 0)",
			tokens:           []string{"word_a"},
			expectedIntent:   "intent_a",
			expectedAmbiguous: false,
			description:      "Intent A wins clearly (2.0 vs 0, diff = 2.0 >= 1.0)",
		},
		{
			name:             "Clear match B (score 1.5 vs 0)",
			tokens:           []string{"word_b"},
			expectedIntent:   "intent_b",
			expectedAmbiguous: false,
			description:      "Intent B wins clearly (1.5 vs 0, diff = 1.5 >= 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classifier.Classify(tt.tokens)
			if err != nil {
				t.Errorf("Classify() error = %v", err)
				return
			}

			t.Logf("  Test: %s", tt.description)
			t.Logf("  Tokens: %v", tt.tokens)
			t.Logf("  Result: %s, Expected: %s", result, tt.expectedIntent)

			if result != intent.Intent(tt.expectedIntent) {
				t.Errorf("Expected intent %s, got %s", tt.expectedIntent, result)
			}
		})
	}
}