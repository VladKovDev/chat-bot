package presenter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func TestValidateActualResponsesAndIntentCatalog(t *testing.T) {
	t.Parallel()

	configPath, err := filepath.Abs(filepath.Join("..", "..", "..", "configs"))
	if err != nil {
		t.Fatalf("config path abs: %v", err)
	}
	p, err := NewPresenter(configPath)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	catalog, err := LoadIntentCatalog(configPath)
	if err != nil {
		t.Fatalf("load intent catalog: %v", err)
	}

	validator := NewValidator(p.GetAll(), logger.Noop())
	if err := validator.Validate(); err != nil {
		t.Fatalf("validate responses: %v", err)
	}
	if err := validator.ValidateCatalog(catalog); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}
}

func TestValidateFailsOnPlaceholderMismatch(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeResponsesFile(t, tempDir, `{
  "booking_found": {
    "message": "Запись {service} {broken_placeholder}",
    "options": ["Назад"]
  }
}`)

	p, err := NewPresenter(tempDir)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	validator := NewValidator(p.GetAll(), logger.Noop())
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected placeholder validation error")
	}
	if !strings.Contains(err.Error(), "placeholder mismatch") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestPresentNormalizesLegacyOptionsIntoTypedQuickReplies(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeResponsesFile(t, tempDir, `{
  "main_menu": {
    "message": "Меню",
    "options": ["Связаться с оператором", "Вернуться в главное меню"]
  }
}`)

	p, err := NewPresenter(tempDir)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	resp, err := p.Present("main_menu", state.StateWaitingForCategory)
	if err != nil {
		t.Fatalf("present: %v", err)
	}
	if len(resp.QuickReplies) != 2 {
		t.Fatalf("quick replies = %#v, want 2 items", resp.QuickReplies)
	}
	if resp.QuickReplies[0].Action != "send_text" {
		t.Fatalf("quick reply action = %q, want %q", resp.QuickReplies[0].Action, "send_text")
	}
	if got := resp.QuickReplies[0].Payload["text"]; got != "Связаться с оператором" {
		t.Fatalf("payload.text = %#v, want operator label", got)
	}
}

func TestValidateCatalogFailsOnMissingResponseAndUnknownAction(t *testing.T) {
	t.Parallel()

	validator := NewValidator(map[string]*ResponseConfig{
		"start": {Message: "Старт"},
	}, logger.Noop())

	err := validator.ValidateCatalog(&IntentCatalog{
		Intents: []IntentDefinition{
			{
				Key:            "greeting",
				Category:       "system",
				ResolutionType: "static_response",
				ResponseKey:    "missing_response",
				Examples:       []string{"привет", "здравствуйте", "добрый день", "добрый вечер", "помогите", "подскажите", "здравствуйте бот", "можно спросить"},
				E2ECoverage:    []string{"E2E-001"},
			},
			{
				Key:            "request_operator",
				Category:       "operator",
				ResolutionType: "operator_handoff",
				ResponseKey:    "start",
				Action:         "missing_action",
				Examples:       []string{"оператор", "человек", "поддержка", "живой оператор", "соедините с человеком", "специалист", "нужен оператор", "передайте в поддержку"},
				E2ECoverage:    []string{"E2E-022"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "references missing response_key") {
		t.Fatalf("expected missing response error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "references unknown action") {
		t.Fatalf("expected unknown action error, got: %v", err)
	}
}

func writeResponsesFile(t *testing.T, dir string, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, "responses.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write responses.json: %v", err)
	}
}
