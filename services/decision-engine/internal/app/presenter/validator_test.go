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
    "quick_replies": [
      { "id": "menu", "label": "Назад", "action": "select_intent", "payload": { "intent": "return_to_menu", "text": "главное меню" } }
    ]
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

func TestValidateFailsOnLegacyOptions(t *testing.T) {
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

	validator := NewValidator(p.GetAll(), logger.Noop())
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected legacy options validation error")
	}
	if !strings.Contains(err.Error(), "legacy options") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestPresentDoesNotMaterializeLegacyOptionsIntoQuickReplies(t *testing.T) {
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
	if len(resp.QuickReplies) != 0 {
		t.Fatalf("quick replies = %#v, want no implicit materialization from legacy options", resp.QuickReplies)
	}
	if len(resp.Options) != 2 {
		t.Fatalf("options = %#v, want original legacy labels preserved for compatibility", resp.Options)
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

func TestActualBookingRetryPromptProvidesOperatorAndMenuQuickReplies(t *testing.T) {
	t.Parallel()

	configPath, err := filepath.Abs(filepath.Join("..", "..", "..", "configs"))
	if err != nil {
		t.Fatalf("config path abs: %v", err)
	}
	p, err := NewPresenter(configPath)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	resp, err := p.Present("booking_request_identifier_retry", state.StateWaitingForIdentifier)
	if err != nil {
		t.Fatalf("present booking_request_identifier_retry: %v", err)
	}
	if !strings.Contains(resp.Text, "BRG-482910") {
		t.Fatalf("retry prompt = %q, want explicit booking number example", resp.Text)
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

	if reply, ok := quickRepliesByID["booking-request-retry-operator"]; !ok || reply.Action != "request_operator" {
		t.Fatalf("operator retry quick reply missing or invalid: %#v", resp.QuickReplies)
	}
	if reply, ok := quickRepliesByID["booking-request-retry-menu"]; !ok || reply.Action != "select_intent" {
		t.Fatalf("menu retry quick reply missing or invalid: %#v", resp.QuickReplies)
	}
}

func TestActualWorkspaceInfoUsesSelectIntentForPrices(t *testing.T) {
	t.Parallel()

	configPath, err := filepath.Abs(filepath.Join("..", "..", "..", "configs"))
	if err != nil {
		t.Fatalf("config path abs: %v", err)
	}
	p, err := NewPresenter(configPath)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	resp, err := p.Present("workspace_info", state.StateWorkspace)
	if err != nil {
		t.Fatalf("present workspace_info: %v", err)
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

	pricesReply, ok := quickRepliesByID["workspace-prices-info"]
	if !ok {
		t.Fatalf("workspace-prices-info quick reply missing: %#v", resp.QuickReplies)
	}
	if pricesReply.Action != "select_intent" {
		t.Fatalf("workspace-prices-info action = %q, want select_intent", pricesReply.Action)
	}
	if got := pricesReply.Payload["intent"]; got != "ask_workspace_prices" {
		t.Fatalf("workspace-prices-info payload.intent = %#v, want ask_workspace_prices", got)
	}
}

func TestActualPaymentAndTechRecoveryRepliesUseExplicitIntentActions(t *testing.T) {
	t.Parallel()

	configPath, err := filepath.Abs(filepath.Join("..", "..", "..", "configs"))
	if err != nil {
		t.Fatalf("config path abs: %v", err)
	}
	p, err := NewPresenter(configPath)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	tests := []struct {
		responseKey string
		replyID     string
		wantIntent  string
		wantState   state.State
	}{
		{
			responseKey: "payment_debited_not_activated",
			replyID:     "payment-activated-wait",
			wantIntent:  "ask_payment_status",
			wantState:   state.StatePayment,
		},
		{
			responseKey: "payment_failed",
			replyID:     "payment-failed-retry",
			wantIntent:  "payment_not_passed",
			wantState:   state.StatePayment,
		},
		{
			responseKey: "tech_site_not_loading",
			replyID:     "tech-site-fixes",
			wantIntent:  "ask_site_problem",
			wantState:   state.StateTechIssue,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.responseKey+"/"+tt.replyID, func(t *testing.T) {
			t.Parallel()

			resp, err := p.Present(tt.responseKey, tt.wantState)
			if err != nil {
				t.Fatalf("present %s: %v", tt.responseKey, err)
			}

			var found *QuickReplyConfig
			for _, quickReply := range resp.QuickReplies {
				if quickReply.ID != tt.replyID {
					continue
				}
				found = &QuickReplyConfig{
					ID:      quickReply.ID,
					Label:   quickReply.Label,
					Action:  quickReply.Action,
					Payload: quickReply.Payload,
					Order:   quickReply.Order,
				}
				break
			}
			if found == nil {
				t.Fatalf("%s quick reply missing: %#v", tt.replyID, resp.QuickReplies)
			}
			if found.Action != "select_intent" {
				t.Fatalf("%s action = %q, want select_intent", tt.replyID, found.Action)
			}
			if got := found.Payload["intent"]; got != tt.wantIntent {
				t.Fatalf("%s payload.intent = %#v, want %s", tt.replyID, got, tt.wantIntent)
			}
		})
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
