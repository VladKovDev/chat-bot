package presenter

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

func TestRenderActualTemplatesWithoutRawPlaceholders(t *testing.T) {
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
		name  string
		input RenderInput
		want  []string
	}{
		{
			name: "booking found",
			input: RenderInput{
				ResponseKey: "booking_found",
				State:       state.StateBooking,
				ActionResults: map[string]processor.ActionResult{
					action.ActionFindBooking: {
						Success: true,
						Data: map[string]any{
							"status":         "found",
							"booking_number": "BRG-482910",
							"service":        "Женская стрижка",
							"master":         "Анна Петрова",
							"date":           "2026-05-15",
							"time":           "14:30",
							"booking_status": "confirmed",
						},
					},
				},
			},
			want: []string{"Женская стрижка", "подтверждена"},
		},
		{
			name: "workspace booking found",
			input: RenderInput{
				ResponseKey: "workspace_booking_found",
				State:       state.StateWorkspace,
				ActionResults: map[string]processor.ActionResult{
					action.ActionFindWorkspaceBooking: {
						Success: true,
						Data: map[string]any{
							"status":         "found",
							"booking_number": "WS-1001",
							"workspace_type": "hot_seat",
							"date":           "2026-05-15",
							"time":           "10:00",
							"duration":       "4",
							"booking_status": "confirmed",
						},
					},
				},
			},
			want: []string{"Горячее место", "подтверждена"},
		},
		{
			name: "payment found",
			input: RenderInput{
				ResponseKey: "payment_found",
				State:       state.StatePayment,
				ActionResults: map[string]processor.ActionResult{
					action.ActionFindPayment: {
						Success: true,
						Data: map[string]any{
							"status":         "found",
							"payment_id":     "PAY-123456",
							"amount":         2000,
							"date":           "2026-05-14T10:15:00Z",
							"payment_status": "completed",
							"purpose":        "Женская стрижка",
						},
					},
				},
			},
			want: []string{"PAY-123456", "оплачен"},
		},
		{
			name: "account found",
			input: RenderInput{
				ResponseKey: "account_found",
				State:       state.StateAccount,
				ActionResults: map[string]processor.ActionResult{
					action.ActionFindUserAccount: {
						Success: true,
						Data: map[string]any{
							"status":         "found",
							"user_id":        "usr-100001",
							"email":          "user1@example.com",
							"phone":          "+7 (999) 123-45-67",
							"account_status": "active",
						},
					},
				},
			},
			want: []string{"usr-100001", "активен"},
		},
		{
			name: "workspace static",
			input: RenderInput{
				ResponseKey: "workspace_types_prices",
				State:       state.StateWorkspace,
			},
			want: []string{"Горячее место", "200"},
		},
		{
			name: "escalation context",
			input: RenderInput{
				ResponseKey: "escalation_context_sent",
				State:       state.StateEscalatedToOperator,
				SessionContext: SessionContext{
					Mode:           session.ModeWaitingOperator,
					OperatorStatus: session.OperatorStatusWaiting,
				},
				Data: map[string]any{"question": "нужен оператор"},
			},
			want: []string{"нужен оператор"},
		},
		{
			name: "fallback canned",
			input: RenderInput{
				ResponseKey: "provider_lookup_unavailable",
				State:       state.StatePayment,
			},
			want: []string{"оператор"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, err := p.Render(tt.input)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			assertNoRawPlaceholders(t, resp.Text)
			for _, want := range tt.want {
				if !strings.Contains(resp.Text, want) {
					t.Fatalf("rendered text = %q, want substring %q", resp.Text, want)
				}
			}
			if resp.QuickReplies == nil && len(resp.Options) > 0 {
				t.Fatalf("quick replies were not materialized as typed objects")
			}
		})
	}
}

func TestRenderMissingPlaceholderUsesSafeCannedFallback(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeResponsesFile(t, tempDir, `{
  "booking_found": {
    "message": "Услуга: {service}, статус: {status}",
    "quick_replies": [
      {"id": "operator", "label": "Оператор", "action": "request_operator"}
    ]
  },
  "error_generic": {
    "message": "Не удалось подготовить ответ. Попробуйте позже или обратитесь к оператору.",
    "quick_replies": [
      {"id": "operator", "label": "Оператор", "action": "request_operator"}
    ]
  }
}`)

	p, err := NewPresenter(tempDir)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	resp, err := p.Render(RenderInput{
		ResponseKey: "booking_found",
		State:       state.StateBooking,
		ActionResults: map[string]processor.ActionResult{
			action.ActionFindBooking: {
				Success: true,
				Data: map[string]any{
					"status":         "found",
					"booking_status": "confirmed",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertNoRawPlaceholders(t, resp.Text)
	if strings.Contains(resp.Text, "Услуга:") {
		t.Fatalf("rendered original template despite missing placeholder: %q", resp.Text)
	}
	if len(resp.QuickReplies) != 1 || resp.QuickReplies[0].Action != "request_operator" {
		t.Fatalf("fallback quick replies = %#v, want typed operator reply", resp.QuickReplies)
	}
}

func TestRenderSupportsDirectTemplateInput(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeResponsesFile(t, tempDir, `{
  "error_generic": {
    "message": "Не удалось подготовить ответ.",
    "options": []
  }
}`)
	p, err := NewPresenter(tempDir)
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	resp, err := p.Render(RenderInput{
		Template: &ResponseConfig{
			Message: "Ваш вопрос: {question}",
			QuickReplies: []QuickReplyConfig{
				{ID: "operator", Label: "Оператор", Action: "request_operator"},
			},
		},
		State: state.StateEscalatedToOperator,
		Data:  map[string]any{"question": "подключите специалиста"},
	})
	if err != nil {
		t.Fatalf("render direct template: %v", err)
	}
	assertNoRawPlaceholders(t, resp.Text)
	if !strings.Contains(resp.Text, "подключите специалиста") {
		t.Fatalf("rendered text = %q, want direct template data", resp.Text)
	}
	if len(resp.QuickReplies) != 1 || resp.QuickReplies[0].ID != "operator" {
		t.Fatalf("quick replies = %#v, want typed direct template reply", resp.QuickReplies)
	}
}

func assertNoRawPlaceholders(t *testing.T, text string) {
	t.Helper()

	for _, placeholder := range []string{"{service}", "{date}", "{status}", "{question}", "{workspace_type}", "{payment_id}", "{user_id}"} {
		if strings.Contains(text, placeholder) {
			t.Fatalf("raw placeholder %s leaked in response: %q", placeholder, text)
		}
	}
	if strings.Contains(text, "{") || strings.Contains(text, "}") {
		t.Fatalf("raw placeholder braces leaked in response: %q", text)
	}
}
