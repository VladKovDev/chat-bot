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
    "options": ["Связаться с оператором", "Вернуться в главное меню", "❓ Не приходит код подтверждения"]
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
	if len(resp.QuickReplies) != 3 {
		t.Fatalf("quick replies = %#v, want 3 items", resp.QuickReplies)
	}
	if resp.QuickReplies[0].Action != "request_operator" {
		t.Fatalf("quick reply action = %q, want %q", resp.QuickReplies[0].Action, "request_operator")
	}
	if resp.QuickReplies[1].Action != "select_intent" {
		t.Fatalf("quick reply action = %q, want %q", resp.QuickReplies[1].Action, "select_intent")
	}
	if got := resp.QuickReplies[1].Payload["intent"]; got != "return_to_menu" {
		t.Fatalf("payload.intent = %#v, want return_to_menu", got)
	}
	if resp.QuickReplies[2].Action != "send_text" {
		t.Fatalf("quick reply action = %q, want %q", resp.QuickReplies[2].Action, "send_text")
	}
	if got := resp.QuickReplies[2].Payload["text"]; got != "Не приходит код подтверждения" {
		t.Fatalf("payload.text = %#v, want sanitized label", got)
	}
}

func TestActualStartMenuUsesIntentQuickRepliesForCategories(t *testing.T) {
	t.Parallel()

	configPath, err := filepath.Abs(filepath.Join("..", "..", "..", "configs"))
	if err != nil {
		t.Fatalf("config path abs: %v", err)
	}
	p, err := NewPresenter(configPath)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	resp, err := p.Present("start", state.StateWaitingForCategory)
	if err != nil {
		t.Fatalf("present start: %v", err)
	}

	quickRepliesByID := make(map[string]map[string]any, len(resp.QuickReplies))
	for _, quickReply := range resp.QuickReplies {
		if quickReply.Action == "select_intent" {
			quickRepliesByID[quickReply.ID] = quickReply.Payload
		}
	}

	if got := quickRepliesByID["menu-account"]["intent"]; got != "ask_account_help" {
		t.Fatalf("menu-account payload.intent = %#v, want ask_account_help", got)
	}
	if got := quickRepliesByID["menu-services"]["intent"]; got != "ask_services_info" {
		t.Fatalf("menu-services payload.intent = %#v, want ask_services_info", got)
	}
}

func TestActualBookingInfoUsesSelectIntentForStatusLookup(t *testing.T) {
	t.Parallel()

	configPath, err := filepath.Abs(filepath.Join("..", "..", "..", "configs"))
	if err != nil {
		t.Fatalf("config path abs: %v", err)
	}
	p, err := NewPresenter(configPath)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	resp, err := p.Present("booking_info", state.StateBooking)
	if err != nil {
		t.Fatalf("present booking_info: %v", err)
	}

	quickRepliesByID := make(map[string]QuickReplyConfig, len(resp.QuickReplies))
	for _, quickReply := range resp.QuickReplies {
		quickRepliesByID[quickReply.ID] = QuickReplyConfig{
			ID:      quickReply.ID,
			Label:   quickReply.Label,
			Action:  quickReply.Action,
			Payload: quickReply.Payload,
			Order:   quickReply.Order,
		}
	}

	statusReply, ok := quickRepliesByID["booking-status-check"]
	if !ok {
		t.Fatalf("booking-status-check quick reply missing: %#v", resp.QuickReplies)
	}
	if statusReply.Action != "select_intent" {
		t.Fatalf("booking-status-check action = %q, want select_intent", statusReply.Action)
	}
	if got := statusReply.Payload["intent"]; got != "ask_booking_status" {
		t.Fatalf("booking-status-check payload.intent = %#v, want ask_booking_status", got)
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
