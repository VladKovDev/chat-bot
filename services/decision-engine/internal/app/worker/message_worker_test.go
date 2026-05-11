package worker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appdecision "github.com/VladKovDev/chat-bot/internal/app/decision"
	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/apperror"
	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	operatorDomain "github.com/VladKovDev/chat-bot/internal/domain/operator"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/google/uuid"
)

func TestHandleMessagePersistsSuccessfulFlowInSingleTransaction(t *testing.T) {
	t.Parallel()

	confidence := 0.91
	sessionRepo := newWorkerSessionRepo()
	persistence := newFakeMessagePersistence(sessionRepo)
	decision := fakeDecisionService{
		result: appdecision.Result{
			Intent:      "request_operator",
			State:       state.StateEscalatedToOperator,
			Topic:       "support",
			ResponseKey: "operator_handoff_requested",
			Actions:     []string{"find_payment"},
			ActionContext: map[string]any{
				"provided_identifier": "PAY-123456",
				"identifier_type":     "payment_id",
			},
			Confidence: &confidence,
			Candidates: []appdecision.Candidate{
				{IntentKey: "request_operator", Confidence: confidence},
				{IntentKey: "ask_payment_status", Confidence: 0.61},
			},
			Event: session.EventRequestOperator,
		},
	}

	proc := processor.NewProcessor(logger.Noop())
	proc.Register("find_payment", auditAction{})
	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		decision,
		proc,
		mustPresenter(t),
		persistence,
		logger.Noop(),
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "позови оператора по оплате PAY-123456",
		Channel:   session.ChannelWebsite,
		ClientID:  "browser-a",
		RequestID: "req-success",
		Timestamp: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	if persistence.begins != 1 || persistence.commits != 1 || persistence.rollbacks != 0 {
		t.Fatalf("transaction counters = begin:%d commit:%d rollback:%d, want 1/1/0",
			persistence.begins, persistence.commits, persistence.rollbacks)
	}
	if resp.UserMessageID == uuid.Nil || resp.BotMessageID == uuid.Nil {
		t.Fatalf("response message ids not populated: user=%s bot=%s", resp.UserMessageID, resp.BotMessageID)
	}

	messages := persistence.messages[resp.SessionID]
	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(messages))
	}
	if messages[0].SenderType != message.SenderTypeUser || messages[1].SenderType != message.SenderTypeBot {
		t.Fatalf("senders = %q/%q, want user/bot", messages[0].SenderType, messages[1].SenderType)
	}

	logs := persistence.decisionLogs[resp.SessionID]
	if len(logs) != 1 {
		t.Fatalf("decision log count = %d, want 1", len(logs))
	}
	if logs[0].MessageID != resp.UserMessageID || logs[0].Intent != "request_operator" {
		t.Fatalf("decision log = %#v, want user message and request_operator", logs[0])
	}
	if len(logs[0].Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(logs[0].Candidates))
	}
	if logs[0].Threshold == nil || *logs[0].Threshold != appdecision.DefaultMatchThreshold {
		t.Fatalf("threshold = %#v, want %v", logs[0].Threshold, appdecision.DefaultMatchThreshold)
	}

	actionLogs := persistence.actionLogs[resp.SessionID]
	if len(actionLogs) != 1 {
		t.Fatalf("action log count = %d, want 1", len(actionLogs))
	}
	audit, ok := actionLogs[0].ResponsePayload["audit"].(map[string]any)
	if !ok {
		t.Fatalf("action audit missing from response payload: %#v", actionLogs[0].ResponsePayload)
	}
	if audit["provider"] != "mock_payment_provider" || audit["status"] != "found" {
		t.Fatalf("action audit = %#v, want safe provider audit", audit)
	}

	transitions := persistence.transitions[resp.SessionID]
	if len(transitions) != 1 {
		t.Fatalf("transition count = %d, want 1", len(transitions))
	}
	if transitions[0].From != session.ModeStandard || transitions[0].To != session.ModeWaitingOperator {
		t.Fatalf("transition = %q -> %q, want standard -> waiting_operator", transitions[0].From, transitions[0].To)
	}

	updated := sessionRepo.sessions[resp.SessionID]
	if updated.Mode != session.ModeWaitingOperator || updated.OperatorStatus != session.OperatorStatusWaiting {
		t.Fatalf("updated mode/operator = %q/%q, want waiting_operator/waiting",
			updated.Mode, updated.OperatorStatus)
	}
	if updated.LastIntent != "request_operator" || updated.ActiveTopic != "support" {
		t.Fatalf("context = intent:%q topic:%q, want request_operator/support",
			updated.LastIntent, updated.ActiveTopic)
	}
}

func TestHandleMessagePersistsRepresentativeBRDGateFlows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		text              string
		clientID          string
		intent            apppresenter.IntentDefinition
		match             appdecision.MatchResult
		registerActions   func(*processor.Processor)
		prepareSession    func(t *testing.T, repo *workerSessionRepo)
		wantResponseKey   string
		wantState         state.State
		wantTopic         string
		wantActions       int
		wantTransitions   int
		wantFallbackCount int
		wantMode          session.Mode
		wantOperator      session.OperatorStatus
		wantTextFragment  string
	}{
		{
			name:     "faq knowledge answer",
			text:     "как записаться на услугу?",
			clientID: "brd-faq",
			intent: apppresenter.IntentDefinition{
				Key:            "ask_faq_booking",
				Category:       "services",
				ResolutionType: "knowledge",
				ResponseKey:    "services_faq_booking",
				Examples:       []string{"как записаться"},
			},
			match: appdecision.MatchResult{
				IntentKey:  "ask_faq_booking",
				Confidence: 0.95,
				Candidates: []appdecision.Candidate{
					{IntentKey: "ask_faq_booking", Confidence: 0.95, Source: appdecision.CandidateSourceIntentExample},
					{IntentKey: "ask_payment_status", Confidence: 0.54, Source: appdecision.CandidateSourceIntentExample},
				},
			},
			wantResponseKey:  "services_faq_booking",
			wantState:        state.StateServices,
			wantTopic:        "services",
			wantMode:         session.ModeStandard,
			wantOperator:     session.OperatorStatusNone,
			wantTextFragment: "Как записаться",
		},
		{
			name:     "business lookup with action audit",
			text:     "проверь оплату PAY-123456",
			clientID: "brd-payment",
			intent: apppresenter.IntentDefinition{
				Key:                 "ask_payment_status",
				Category:            "payment",
				ResolutionType:      "business_lookup",
				ResponseKey:         "payment_request_id",
				FallbackResponseKey: "payment_request_id",
				Action:              action.ActionFindPayment,
				Examples:            []string{"проверь оплату"},
			},
			match: appdecision.MatchResult{
				IntentKey:  "ask_payment_status",
				Confidence: 0.93,
				Candidates: []appdecision.Candidate{
					{IntentKey: "ask_payment_status", Confidence: 0.93, Source: appdecision.CandidateSourceIntentExample},
					{IntentKey: "ask_faq_booking", Confidence: 0.45, Source: appdecision.CandidateSourceIntentExample},
				},
			},
			registerActions: func(proc *processor.Processor) {
				proc.Register(action.ActionFindPayment, paymentFoundAction{})
			},
			wantResponseKey:  "payment_found",
			wantState:        state.StatePayment,
			wantTopic:        "payment",
			wantActions:      1,
			wantMode:         session.ModeStandard,
			wantOperator:     session.OperatorStatusNone,
			wantTextFragment: "PAY-123456",
		},
		{
			name:     "unknown repeated low confidence escalates",
			text:     "вообще непонятный вопрос без категории",
			clientID: "brd-unknown",
			intent: apppresenter.IntentDefinition{
				Key:            "unknown",
				Category:       "fallback",
				ResolutionType: "fallback",
				ResponseKey:    "clarify_request",
				Examples:       []string{"не знаю"},
			},
			match: appdecision.MatchResult{
				Confidence: 0.18,
				Candidates: []appdecision.Candidate{
					{IntentKey: "unknown", Confidence: 0.18, Source: appdecision.CandidateSourceIntentExample},
					{IntentKey: "ask_payment_status", Confidence: 0.16, Source: appdecision.CandidateSourceIntentExample},
				},
			},
			registerActions: func(proc *processor.Processor) {
				proc.Register(action.ActionEscalateToOperator, operatorAuditAction{})
			},
			prepareSession: func(t *testing.T, repo *workerSessionRepo) {
				t.Helper()
				started, err := session.NewService(repo).StartSession(context.Background(), session.Identity{
					Channel:  session.ChannelWebsite,
					ClientID: "brd-unknown",
				})
				if err != nil {
					t.Fatalf("start existing low-confidence session: %v", err)
				}
				existing := started.Session
				existing.FallbackCount = 1
				repo.set(existing)
			},
			wantResponseKey:   "operator_handoff_requested",
			wantState:         state.StateEscalatedToOperator,
			wantActions:       1,
			wantTransitions:   1,
			wantFallbackCount: 2,
			wantMode:          session.ModeWaitingOperator,
			wantOperator:      session.OperatorStatusWaiting,
			wantTextFragment:  "Подключаю оператора",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sessionRepo := newWorkerSessionRepo()
			if tt.prepareSession != nil {
				tt.prepareSession(t, sessionRepo)
			}
			persistence := newFakeMessagePersistence(sessionRepo)
			proc := processor.NewProcessor(logger.Noop())
			if tt.registerActions != nil {
				tt.registerActions(proc)
			}
			decisionService, err := appdecision.NewService(&apppresenter.IntentCatalog{
				Intents: []apppresenter.IntentDefinition{tt.intent},
			}, controlledMatcher{result: tt.match}, logger.Noop())
			if err != nil {
				t.Fatalf("new decision service: %v", err)
			}

			worker := NewMessageWorker(
				session.NewService(sessionRepo),
				decisionService,
				proc,
				mustPresenter(t),
				persistence,
				logger.Noop(),
			)

			resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
				Text:      tt.text,
				Channel:   session.ChannelWebsite,
				ClientID:  tt.clientID,
				RequestID: "req-" + tt.clientID,
				Timestamp: time.Date(2026, 5, 10, 13, 0, 0, 0, time.UTC),
			})
			if err != nil {
				t.Fatalf("HandleMessage returned error: %v", err)
			}

			if persistence.begins != 1 || persistence.commits != 1 || persistence.rollbacks != 0 {
				t.Fatalf("transaction counters = begin:%d commit:%d rollback:%d, want 1/1/0",
					persistence.begins, persistence.commits, persistence.rollbacks)
			}

			messages := persistence.messages[resp.SessionID]
			if len(messages) != 2 {
				t.Fatalf("message count = %d, want inbound and outbound", len(messages))
			}
			if messages[0].SenderType != message.SenderTypeUser || messages[0].Text != tt.text {
				t.Fatalf("inbound message = %#v, want persisted user text", messages[0])
			}
			if messages[1].SenderType != message.SenderTypeBot || !strings.Contains(messages[1].Text, tt.wantTextFragment) {
				t.Fatalf("outbound message = %#v, want bot text containing %q", messages[1], tt.wantTextFragment)
			}
			if resp.UserMessageID != messages[0].ID || resp.BotMessageID != messages[1].ID {
				t.Fatalf("response ids user/bot=%s/%s, stored=%s/%s",
					resp.UserMessageID, resp.BotMessageID, messages[0].ID, messages[1].ID)
			}

			logs := persistence.decisionLogs[resp.SessionID]
			if len(logs) != 1 {
				t.Fatalf("decision log count = %d, want 1", len(logs))
			}
			if logs[0].MessageID != resp.UserMessageID ||
				logs[0].Intent != firstNonEmptyString(tt.match.IntentKey, "unknown") ||
				logs[0].State != tt.wantState ||
				logs[0].ResponseKey != tt.wantResponseKey {
				t.Fatalf("decision log = %#v", logs[0])
			}
			if len(logs[0].Candidates) != len(tt.match.Candidates) {
				t.Fatalf("candidate count = %d, want %d", len(logs[0].Candidates), len(tt.match.Candidates))
			}
			if logs[0].Confidence == nil || *logs[0].Confidence != tt.match.Confidence {
				t.Fatalf("confidence = %#v, want %v", logs[0].Confidence, tt.match.Confidence)
			}

			actionLogs := persistence.actionLogs[resp.SessionID]
			if len(actionLogs) != tt.wantActions {
				t.Fatalf("action log count = %d, want %d", len(actionLogs), tt.wantActions)
			}
			if tt.wantActions > 0 && actionLogs[0].ResponsePayload["audit"] == nil {
				t.Fatalf("action log audit missing: %#v", actionLogs[0].ResponsePayload)
			}

			transitions := persistence.transitions[resp.SessionID]
			if len(transitions) != tt.wantTransitions {
				t.Fatalf("transition count = %d, want %d", len(transitions), tt.wantTransitions)
			}
			if tt.wantTransitions > 0 &&
				(transitions[0].From != session.ModeStandard || transitions[0].To != session.ModeWaitingOperator) {
				t.Fatalf("transition = %q -> %q, want standard -> waiting_operator",
					transitions[0].From, transitions[0].To)
			}

			updated := sessionRepo.sessions[resp.SessionID]
			if updated.State != tt.wantState ||
				updated.ActiveTopic != tt.wantTopic ||
				updated.LastIntent != firstNonEmptyString(tt.match.IntentKey, "unknown") ||
				updated.FallbackCount != tt.wantFallbackCount ||
				updated.Mode != tt.wantMode ||
				updated.OperatorStatus != tt.wantOperator {
				t.Fatalf("session context = %#v", updated)
			}
			if resp.State != tt.wantState || resp.Mode != tt.wantMode || resp.OperatorStatus != tt.wantOperator {
				t.Fatalf("response context state/mode/operator = %q/%q/%q",
					resp.State, resp.Mode, resp.OperatorStatus)
			}
		})
	}
}

func TestHandleMessageQueuesManualOperatorRequest(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	persistence := newFakeMessagePersistence(sessionRepo)
	proc := processor.NewProcessor(logger.Noop())
	proc.Register(action.ActionEscalateToOperator, operatorAuditAction{})
	handoff := newFakeWorkerHandoff(sessionRepo)
	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		fakeDecisionService{
			result: appdecision.Result{
				Intent:      "request_operator",
				State:       state.StateEscalatedToOperator,
				ResponseKey: "operator_handoff_requested",
				Actions:     []string{action.ActionEscalateToOperator},
				ActionContext: map[string]any{
					"handoff_reason": "manual_request",
				},
				Event: session.EventRequestOperator,
			},
		},
		proc,
		mustPresenter(t),
		persistence,
		logger.Noop(),
		handoff,
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "оператор",
		Channel:   session.ChannelWebsite,
		ClientID:  "manual-user",
		RequestID: "req-manual",
		Timestamp: time.Date(2026, 5, 10, 12, 10, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	if resp.Mode != session.ModeWaitingOperator || resp.OperatorStatus != session.OperatorStatusWaiting {
		t.Fatalf("response mode/operator = %q/%q, want waiting_operator/waiting", resp.Mode, resp.OperatorStatus)
	}
	if len(handoff.items) != 1 || handoff.items[0].Reason != operatorDomain.ReasonManualRequest {
		t.Fatalf("handoff items = %+v, want manual_request", handoff.items)
	}
	if got := len(handoff.transitions[resp.SessionID]); got != 1 {
		t.Fatalf("handoff transition count = %d, want 1", got)
	}
	actionLogs := persistence.actionLogs[resp.SessionID]
	if len(actionLogs) != 1 {
		t.Fatalf("action log count = %d, want escalation action log", len(actionLogs))
	}
	if actionLogs[0].ActionType != action.ActionEscalateToOperator {
		t.Fatalf("action type = %q, want escalate_to_operator", actionLogs[0].ActionType)
	}
}

func TestHandleMessageQueuesComplaintWithComplaintReason(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	persistence := newFakeMessagePersistence(sessionRepo)
	proc := processor.NewProcessor(logger.Noop())
	proc.Register(action.ActionEscalateToOperator, operatorAuditAction{})
	handoff := newFakeWorkerHandoff(sessionRepo)
	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		fakeDecisionService{
			result: appdecision.Result{
				Intent:      "report_complaint",
				State:       state.StateComplaint,
				Topic:       "complaint",
				ResponseKey: "complaint_info_collected",
				Actions:     []string{action.ActionEscalateToOperator},
				ActionContext: map[string]any{
					"handoff_reason": "complaint",
				},
				Event: session.EventRequestOperator,
			},
		},
		proc,
		mustPresenter(t),
		persistence,
		logger.Noop(),
		handoff,
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "хочу пожаловаться",
		Channel:   session.ChannelWebsite,
		ClientID:  "complaint-user",
		RequestID: "req-complaint",
		Timestamp: time.Date(2026, 5, 10, 12, 11, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	if len(handoff.items) != 1 || handoff.items[0].Reason != operatorDomain.ReasonComplaint {
		t.Fatalf("handoff items = %+v, want complaint", handoff.items)
	}
	if handoff.items[0].ContextSnapshot.ActiveTopic != "complaint" ||
		handoff.items[0].ContextSnapshot.LastIntent != "report_complaint" {
		t.Fatalf("snapshot = %+v, want complaint/report_complaint", handoff.items[0].ContextSnapshot)
	}
	if resp.Mode != session.ModeWaitingOperator {
		t.Fatalf("response mode = %q, want waiting_operator", resp.Mode)
	}
}

func TestHandleMessageQueuesRepeatedLowConfidence(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	started, err := session.NewService(sessionRepo).StartSession(context.Background(), session.Identity{
		Channel:  session.ChannelWebsite,
		ClientID: "fallback-user",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	existing := started.Session
	existing.FallbackCount = 1
	sessionRepo.set(existing)

	persistence := newFakeMessagePersistence(sessionRepo)
	proc := processor.NewProcessor(logger.Noop())
	proc.Register(action.ActionEscalateToOperator, operatorAuditAction{})
	handoff := newFakeWorkerHandoff(sessionRepo)
	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		fakeDecisionService{
			result: appdecision.Result{
				Intent:        "unknown",
				State:         state.StateEscalatedToOperator,
				ResponseKey:   "operator_handoff_requested",
				Actions:       []string{action.ActionEscalateToOperator},
				ActionContext: map[string]any{"handoff_reason": "low_confidence_repeated"},
				LowConfidence: true,
				Event:         session.EventRequestOperator,
			},
		},
		proc,
		mustPresenter(t),
		persistence,
		logger.Noop(),
		handoff,
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "эээ не знаю",
		Channel:   session.ChannelWebsite,
		ClientID:  "fallback-user",
		RequestID: "req-fallback",
		Timestamp: time.Date(2026, 5, 10, 12, 12, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	if len(handoff.items) != 1 || handoff.items[0].Reason != operatorDomain.ReasonLowConfidenceRepeated {
		t.Fatalf("handoff items = %+v, want low_confidence_repeated", handoff.items)
	}
	if got := handoff.sessions.sessions[resp.SessionID].FallbackCount; got != 2 {
		t.Fatalf("fallback_count = %d, want 2", got)
	}
}

func TestHandleMessageQueuesProviderUnavailableAsBusinessError(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	persistence := newFakeMessagePersistence(sessionRepo)
	proc := processor.NewProcessor(logger.Noop())
	proc.Register(action.ActionFindPayment, unavailableAction{})
	handoff := newFakeWorkerHandoff(sessionRepo)
	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		fakeDecisionService{
			result: appdecision.Result{
				Intent:                  "ask_payment_status",
				State:                   state.StatePayment,
				Topic:                   "payment",
				ResponseKey:             "payment_request_id",
				Actions:                 []string{action.ActionFindPayment},
				ActionContext:           map[string]any{"provided_identifier": "PAY-ERROR-503", "identifier_type": "payment_id"},
				Event:                   session.EventMessageReceived,
				UseActionResponseSelect: true,
			},
		},
		proc,
		mustPresenter(t),
		persistence,
		logger.Noop(),
		handoff,
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "проверь оплату PAY-ERROR-503",
		Channel:   session.ChannelWebsite,
		ClientID:  "provider-user",
		RequestID: "req-provider",
		Timestamp: time.Date(2026, 5, 10, 12, 13, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	if len(handoff.items) != 1 || handoff.items[0].Reason != operatorDomain.ReasonBusinessError {
		t.Fatalf("handoff items = %+v, want business_error", handoff.items)
	}
	if resp.Mode != session.ModeWaitingOperator {
		t.Fatalf("mode = %q, want waiting_operator", resp.Mode)
	}
	if resp.Text == "" || resp.Text == "payment provider unavailable" {
		t.Fatalf("unsafe provider response text = %q", resp.Text)
	}
	actionLogs := persistence.actionLogs[resp.SessionID]
	if len(actionLogs) != 1 {
		t.Fatalf("action log count = %d, want provider action log", len(actionLogs))
	}
	audit, ok := actionLogs[0].ResponsePayload["audit"].(map[string]any)
	if !ok || audit["status"] != "unavailable" || audit["error_code"] != "provider_unavailable" {
		t.Fatalf("provider audit = %#v, want unavailable/provider_unavailable", actionLogs[0].ResponsePayload["audit"])
	}
}

func TestHandleMessageRendersSelectedActionResponseData(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	persistence := newFakeMessagePersistence(sessionRepo)
	proc := processor.NewProcessor(logger.Noop())
	proc.Register(action.ActionFindPayment, paymentFoundAction{})
	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		fakeDecisionService{
			result: appdecision.Result{
				Intent:                  "ask_payment_status",
				State:                   state.StatePayment,
				Topic:                   "payment",
				ResponseKey:             "payment_request_id",
				Actions:                 []string{action.ActionFindPayment},
				ActionContext:           map[string]any{"provided_identifier": "PAY-123456", "identifier_type": "payment_id"},
				Event:                   session.EventMessageReceived,
				UseActionResponseSelect: true,
			},
		},
		proc,
		mustPresenter(t),
		persistence,
		logger.Noop(),
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "проверь оплату PAY-123456",
		Channel:   session.ChannelWebsite,
		ClientID:  "payment-user",
		RequestID: "req-payment-found",
		Timestamp: time.Date(2026, 5, 10, 12, 14, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	for _, forbidden := range []string{"{payment_id}", "{amount}", "{date}", "{status}", "{purpose}"} {
		if strings.Contains(resp.Text, forbidden) {
			t.Fatalf("raw placeholder %s leaked in response: %q", forbidden, resp.Text)
		}
	}
	if !strings.Contains(resp.Text, "PAY-123456") || !strings.Contains(resp.Text, "оплачен") {
		t.Fatalf("rendered payment response = %q, want action data and localized status", resp.Text)
	}
	if len(resp.QuickReplies) == 0 {
		t.Fatalf("typed quick replies are empty for payment_found")
	}
}

func TestHandleMessagePersistsQuickReplyIDAndUsesTypedSelection(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	persistence := newFakeMessagePersistence(sessionRepo)
	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		fakeDecisionService{
			result: appdecision.Result{
				Intent:      "return_to_menu",
				State:       state.StateWaitingForCategory,
				Topic:       "general",
				ResponseKey: "main_menu",
				Event:       session.EventMessageReceived,
			},
		},
		processor.NewProcessor(logger.Noop()),
		mustPresenter(t),
		persistence,
		logger.Noop(),
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "return_to_menu",
		Channel:   session.ChannelWebsite,
		ClientID:  "quick-reply-user",
		RequestID: "req-quick-reply",
		Timestamp: time.Date(2026, 5, 10, 12, 15, 0, 0, time.UTC),
		QuickReply: &contracts.QuickReplySelection{
			ID:     "renamed-menu",
			Label:  "Changed label",
			Action: "select_intent",
			Payload: map[string]any{
				"intent": "return_to_menu",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	messages := persistence.messages[resp.SessionID]
	if len(messages) < 1 {
		t.Fatalf("messages are empty")
	}
	if messages[0].Intent == nil || *messages[0].Intent != "renamed-menu" {
		t.Fatalf("stored quick reply id = %#v, want renamed-menu", messages[0].Intent)
	}
}

func TestHandleMessageRollsBackWhenMandatoryBotMessageWriteFails(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	persistence := newFakeMessagePersistence(sessionRepo)
	persistence.failBotMessage = true

	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		fakeDecisionService{
			result: appdecision.Result{
				Intent:      "greeting",
				State:       state.StateWaitingForCategory,
				Topic:       "general",
				ResponseKey: "greeting",
				Event:       session.EventGreeting,
			},
		},
		processor.NewProcessor(logger.Noop()),
		mustPresenter(t),
		persistence,
		logger.Noop(),
	)

	_, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		Text:      "привет",
		Channel:   session.ChannelWebsite,
		ClientID:  "browser-b",
		RequestID: "req-fail",
		Timestamp: time.Date(2026, 5, 10, 12, 5, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("HandleMessage returned nil error, want controlled persistence error")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeDatabaseUnavailable {
		t.Fatalf("error = %v, want app database_unavailable error", err)
	}

	if persistence.begins != 1 || persistence.commits != 0 || persistence.rollbacks != 1 {
		t.Fatalf("transaction counters = begin:%d commit:%d rollback:%d, want 1/0/1",
			persistence.begins, persistence.commits, persistence.rollbacks)
	}
	for sessionID, messages := range persistence.messages {
		if len(messages) != 0 {
			t.Fatalf("session %s committed %d messages after rollback, want 0", sessionID, len(messages))
		}
	}
	if len(persistence.decisionLogs) != 0 || len(persistence.actionLogs) != 0 || len(persistence.transitions) != 0 {
		t.Fatalf("logs committed after rollback: decisions=%d actions=%d transitions=%d",
			len(persistence.decisionLogs), len(persistence.actionLogs), len(persistence.transitions))
	}
}

func TestHandleMessagePersistsUserMessageOnlyWhenOperatorConnected(t *testing.T) {
	t.Parallel()

	sessionRepo := newWorkerSessionRepo()
	sessionID := uuid.New()
	sessionRepo.set(session.Session{
		ID:             sessionID,
		UserID:         uuid.New(),
		Channel:        session.ChannelWebsite,
		ClientID:       "browser-operator",
		State:          state.StateEscalatedToOperator,
		Mode:           session.ModeOperatorConnected,
		OperatorStatus: session.OperatorStatusConnected,
		ActiveTopic:    "payment",
		LastIntent:     "request_operator",
		Status:         session.StatusActive,
		Metadata:       map[string]interface{}{"handoff_id": "handoff-a"},
	})
	persistence := newFakeMessagePersistence(sessionRepo)

	worker := NewMessageWorker(
		session.NewService(sessionRepo),
		failingDecisionService{},
		processor.NewProcessor(logger.Noop()),
		mustPresenter(t),
		persistence,
		logger.Noop(),
	)

	resp, err := worker.HandleMessage(context.Background(), contracts.IncomingMessage{
		SessionID: sessionID,
		Text:      "Я отправил номер платежа",
		Channel:   session.ChannelWebsite,
		ClientID:  "browser-operator",
		RequestID: "req-operator-connected",
		Timestamp: time.Date(2026, 5, 10, 12, 15, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	if resp.BotMessageID != uuid.Nil || resp.Text != "" {
		t.Fatalf("operator-connected response bot=%s text=%q, want no bot response", resp.BotMessageID, resp.Text)
	}
	if resp.Mode != session.ModeOperatorConnected || resp.OperatorStatus != session.OperatorStatusConnected {
		t.Fatalf("response mode/operator = %q/%q", resp.Mode, resp.OperatorStatus)
	}

	messages := persistence.messages[sessionID]
	if len(messages) != 1 || messages[0].SenderType != message.SenderTypeUser {
		t.Fatalf("messages = %+v, want one persisted user message", messages)
	}
	if len(persistence.decisionLogs) != 0 || len(persistence.actionLogs) != 0 {
		t.Fatalf("decision/action logs written while operator connected: decisions=%d actions=%d",
			len(persistence.decisionLogs), len(persistence.actionLogs))
	}
}

type fakeDecisionService struct {
	result appdecision.Result
	err    error
}

func (f fakeDecisionService) Decide(
	_ context.Context,
	_ session.Session,
	_ []message.Message,
	_ string,
) (appdecision.Result, error) {
	return f.result, f.err
}

func (f fakeDecisionService) DecideQuickReply(
	_ context.Context,
	_ session.Session,
	_ []message.Message,
	_ appdecision.QuickReplySelection,
	_ string,
) (appdecision.Result, error) {
	return f.result, f.err
}

type failingDecisionService struct{}

func (failingDecisionService) Decide(
	_ context.Context,
	_ session.Session,
	_ []message.Message,
	_ string,
) (appdecision.Result, error) {
	return appdecision.Result{}, errors.New("decision service should not be called")
}

type auditAction struct{}

func (auditAction) Execute(_ context.Context, data action.ActionData) error {
	data.Context["action_result"] = map[string]any{
		"status": "found",
		"found":  true,
		"source": "mock_external",
	}
	data.Context["action_audit"] = map[string]any{
		"provider":    "mock_payment_provider",
		"source":      "mock_external",
		"status":      "found",
		"duration_ms": int64(12),
	}
	return nil
}

type operatorAuditAction struct{}

func (operatorAuditAction) Execute(_ context.Context, data action.ActionData) error {
	reason, _ := data.Context["handoff_reason"].(string)
	if reason == "" {
		reason = "manual_request"
	}
	data.Context["action_result"] = map[string]any{
		"status": "queued_requested",
		"reason": reason,
		"source": "operator_queue",
	}
	data.Context["action_audit"] = map[string]any{
		"provider": "operator_queue",
		"source":   "internal",
		"status":   "queued_requested",
		"reason":   reason,
	}
	return nil
}

type unavailableAction struct{}

func (unavailableAction) Execute(_ context.Context, data action.ActionData) error {
	data.Context["action_result"] = map[string]any{
		"status":     "unavailable",
		"found":      false,
		"source":     "mock_external",
		"error_code": "provider_unavailable",
	}
	data.Context["action_audit"] = map[string]any{
		"provider":    "mock_payment_provider",
		"source":      "mock_external",
		"status":      "unavailable",
		"duration_ms": int64(12),
		"error_code":  "provider_unavailable",
	}
	return nil
}

type paymentFoundAction struct{}

func (paymentFoundAction) Execute(_ context.Context, data action.ActionData) error {
	data.Context["action_result"] = map[string]any{
		"status":         "found",
		"found":          true,
		"source":         "mock_external",
		"payment_id":     "PAY-123456",
		"amount":         2000,
		"date":           "2026-05-14T10:15:00Z",
		"payment_status": "completed",
		"purpose":        "Женская стрижка",
	}
	data.Context["action_audit"] = map[string]any{
		"provider":    "mock_payment_provider",
		"source":      "mock_external",
		"status":      "found",
		"duration_ms": int64(12),
	}
	return nil
}

type fakeWorkerHandoff struct {
	sessions    *workerSessionRepo
	items       []operatorDomain.QueueItem
	transitions map[uuid.UUID][]session.ModeTransition
}

func newFakeWorkerHandoff(sessions *workerSessionRepo) *fakeWorkerHandoff {
	return &fakeWorkerHandoff{
		sessions:    sessions,
		transitions: make(map[uuid.UUID][]session.ModeTransition),
	}
}

func (f *fakeWorkerHandoff) QueueWithDecision(
	_ context.Context,
	sessionID uuid.UUID,
	reason operatorDomain.Reason,
	snapshot operatorDomain.ContextSnapshot,
	decision session.ContextDecision,
) (operatorDomain.QueueItem, error) {
	sess, err := f.sessions.GetByID(context.Background(), sessionID)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}

	queueID := uuid.New()
	if decision.Metadata == nil {
		decision.Metadata = map[string]interface{}{}
	}
	decision.Event = session.EventRequestOperator
	decision.Metadata["handoff_id"] = queueID.String()
	decision.Metadata["handoff_reason"] = string(reason)
	next, transition, err := session.PrepareContextUpdate(&sess, decision)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}
	f.sessions.set(next)
	if transition != nil {
		f.transitions[sessionID] = append(f.transitions[sessionID], *transition)
	}

	item := operatorDomain.QueueItem{
		ID:              queueID,
		SessionID:       sessionID,
		UserID:          next.UserID,
		Status:          operatorDomain.QueueStatusWaiting,
		Reason:          reason,
		ContextSnapshot: snapshot,
		CreatedAt:       time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	f.items = append(f.items, item)
	return item, nil
}

func mustPresenter(t *testing.T) *apppresenter.Presenter {
	t.Helper()
	p, err := apppresenter.NewPresenter("../../../configs")
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}
	return p
}

type workerSessionRepo struct {
	sessions   map[uuid.UUID]session.Session
	byIdentity map[string]uuid.UUID
}

func newWorkerSessionRepo() *workerSessionRepo {
	return &workerSessionRepo{
		sessions:   make(map[uuid.UUID]session.Session),
		byIdentity: make(map[string]uuid.UUID),
	}
}

func (r *workerSessionRepo) Create(_ context.Context, sess session.Session) (session.Session, error) {
	if sess.ID == uuid.Nil {
		sess.ID = uuid.New()
	}
	if sess.UserID == uuid.Nil {
		sess.UserID = uuid.New()
	}
	if sess.Metadata == nil {
		sess.Metadata = map[string]interface{}{}
	}
	r.set(sess)
	return sess, nil
}

func (r *workerSessionRepo) GetByID(_ context.Context, id uuid.UUID) (session.Session, error) {
	sess, ok := r.sessions[id]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	return sess, nil
}

func (r *workerSessionRepo) GetActiveByIdentity(_ context.Context, identity session.Identity) (session.Session, error) {
	id, ok := r.byIdentity[workerIdentityKey(identity)]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	return r.sessions[id], nil
}

func (r *workerSessionRepo) GetByUserID(_ context.Context, userID uuid.UUID, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0)
	for _, sess := range r.sessions {
		if sess.UserID == userID {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (r *workerSessionRepo) Update(_ context.Context, sess session.Session) (session.Session, error) {
	r.set(sess)
	return sess, nil
}

func (r *workerSessionRepo) UpdateContext(_ context.Context, sess session.Session, _ *session.ModeTransition) (session.Session, error) {
	r.set(sess)
	return sess, nil
}

func (r *workerSessionRepo) UpdateState(_ context.Context, id uuid.UUID, st state.State) (session.Session, error) {
	sess, err := r.GetByID(context.Background(), id)
	if err != nil {
		return session.Session{}, err
	}
	sess.State = st
	r.set(sess)
	return sess, nil
}

func (r *workerSessionRepo) UpdateStateWithVersion(ctx context.Context, id uuid.UUID, st state.State) (session.Session, error) {
	sess, err := r.UpdateState(ctx, id, st)
	if err != nil {
		return session.Session{}, err
	}
	sess.Version++
	r.set(sess)
	return sess, nil
}

func (r *workerSessionRepo) UpdateStatus(_ context.Context, id uuid.UUID, status session.Status) (session.Session, error) {
	sess, err := r.GetByID(context.Background(), id)
	if err != nil {
		return session.Session{}, err
	}
	sess.Status = status
	r.set(sess)
	return sess, nil
}

func (r *workerSessionRepo) List(_ context.Context, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0, len(r.sessions))
	for _, sess := range r.sessions {
		result = append(result, sess)
	}
	return result, nil
}

func (r *workerSessionRepo) ListByState(_ context.Context, st state.State, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0)
	for _, sess := range r.sessions {
		if sess.State == st {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (r *workerSessionRepo) ListByStatus(_ context.Context, status session.Status, _ int32, _ int32) ([]session.Session, error) {
	result := make([]session.Session, 0)
	for _, sess := range r.sessions {
		if sess.Status == status {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (r *workerSessionRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.sessions, id)
	return nil
}

func (r *workerSessionRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.sessions)), nil
}

func (r *workerSessionRepo) set(sess session.Session) {
	if sess.Status == "" {
		sess.Status = session.StatusActive
	}
	if sess.Mode == "" {
		sess.Mode = session.ModeStandard
	}
	if sess.OperatorStatus == "" {
		sess.OperatorStatus = session.OperatorStatusNone
	}
	r.sessions[sess.ID] = sess
	r.byIdentity[workerIdentityKey(session.Identity{
		Channel:        sess.Channel,
		ExternalUserID: sess.ExternalUserID,
		ClientID:       sess.ClientID,
	})] = sess.ID
}

func workerIdentityKey(identity session.Identity) string {
	return identity.Channel + "|" + identity.ExternalUserID + "|" + identity.ClientID
}

type fakeMessagePersistence struct {
	sessionRepo *workerSessionRepo

	messages     map[uuid.UUID][]message.Message
	decisionLogs map[uuid.UUID][]DecisionLog
	actionLogs   map[uuid.UUID][]action.Log
	transitions  map[uuid.UUID][]session.ModeTransition

	begins    int
	commits   int
	rollbacks int

	failBotMessage bool
}

func newFakeMessagePersistence(sessionRepo *workerSessionRepo) *fakeMessagePersistence {
	return &fakeMessagePersistence{
		sessionRepo:  sessionRepo,
		messages:     make(map[uuid.UUID][]message.Message),
		decisionLogs: make(map[uuid.UUID][]DecisionLog),
		actionLogs:   make(map[uuid.UUID][]action.Log),
		transitions:  make(map[uuid.UUID][]session.ModeTransition),
	}
}

func (p *fakeMessagePersistence) WithinMessageTransaction(ctx context.Context, fn func(context.Context, MessageTransaction) error) error {
	p.begins++
	tx := &fakeMessageTx{
		parent:        p,
		messages:      cloneMessages(p.messages),
		decisionLogs:  cloneDecisionLogs(p.decisionLogs),
		actionLogs:    cloneActionLogs(p.actionLogs),
		transitions:   cloneTransitions(p.transitions),
		sessions:      cloneSessions(p.sessionRepo.sessions),
		dirtySessions: make(map[uuid.UUID]session.Session),
	}
	if err := fn(ctx, tx); err != nil {
		p.rollbacks++
		return err
	}
	p.messages = tx.messages
	p.decisionLogs = tx.decisionLogs
	p.actionLogs = tx.actionLogs
	p.transitions = tx.transitions
	for sessionID, sess := range tx.dirtySessions {
		p.sessionRepo.sessions[sessionID] = sess
	}
	p.commits++
	return nil
}

type fakeMessageTx struct {
	parent        *fakeMessagePersistence
	messages      map[uuid.UUID][]message.Message
	decisionLogs  map[uuid.UUID][]DecisionLog
	actionLogs    map[uuid.UUID][]action.Log
	transitions   map[uuid.UUID][]session.ModeTransition
	sessions      map[uuid.UUID]session.Session
	dirtySessions map[uuid.UUID]session.Session
}

func (tx *fakeMessageTx) CreateMessage(_ context.Context, msg message.Message) (message.Message, error) {
	if tx.parent.failBotMessage && msg.SenderType == message.SenderTypeBot {
		return message.Message{}, errors.New("forced bot message failure")
	}
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	tx.messages[msg.SessionID] = append(tx.messages[msg.SessionID], msg)
	return msg, nil
}

func (tx *fakeMessageTx) GetLastMessagesBySessionID(_ context.Context, sessionID uuid.UUID, limit int32) ([]message.Message, error) {
	items := tx.messages[sessionID]
	if int(limit) < len(items) {
		items = items[len(items)-int(limit):]
	}
	result := make([]message.Message, len(items))
	copy(result, items)
	return result, nil
}

func (tx *fakeMessageTx) LogDecision(_ context.Context, entry DecisionLog) error {
	tx.decisionLogs[entry.SessionID] = append(tx.decisionLogs[entry.SessionID], entry)
	return nil
}

func (tx *fakeMessageTx) LogAction(_ context.Context, entry action.Log) error {
	tx.actionLogs[entry.SessionID] = append(tx.actionLogs[entry.SessionID], entry)
	return nil
}

func (tx *fakeMessageTx) ApplyContextDecision(
	_ context.Context,
	sess *session.Session,
	decision session.ContextDecision,
) (session.Session, error) {
	next, transition, err := session.PrepareContextUpdate(sess, decision)
	if err != nil {
		return session.Session{}, err
	}
	tx.sessions[next.ID] = next
	tx.dirtySessions[next.ID] = next
	if transition != nil {
		tx.transitions[next.ID] = append(tx.transitions[next.ID], *transition)
	}
	*sess = next
	return next, nil
}

func cloneMessages(src map[uuid.UUID][]message.Message) map[uuid.UUID][]message.Message {
	dst := make(map[uuid.UUID][]message.Message, len(src))
	for key, value := range src {
		dst[key] = append([]message.Message(nil), value...)
	}
	return dst
}

func cloneDecisionLogs(src map[uuid.UUID][]DecisionLog) map[uuid.UUID][]DecisionLog {
	dst := make(map[uuid.UUID][]DecisionLog, len(src))
	for key, value := range src {
		dst[key] = append([]DecisionLog(nil), value...)
	}
	return dst
}

func cloneActionLogs(src map[uuid.UUID][]action.Log) map[uuid.UUID][]action.Log {
	dst := make(map[uuid.UUID][]action.Log, len(src))
	for key, value := range src {
		dst[key] = append([]action.Log(nil), value...)
	}
	return dst
}

func cloneTransitions(src map[uuid.UUID][]session.ModeTransition) map[uuid.UUID][]session.ModeTransition {
	dst := make(map[uuid.UUID][]session.ModeTransition, len(src))
	for key, value := range src {
		dst[key] = append([]session.ModeTransition(nil), value...)
	}
	return dst
}

func cloneSessions(src map[uuid.UUID]session.Session) map[uuid.UUID]session.Session {
	dst := make(map[uuid.UUID]session.Session, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

var _ MessagePersistence = (*fakeMessagePersistence)(nil)

var _ MessageTransaction = (*fakeMessageTx)(nil)

var _ DecisionService = fakeDecisionService{}

var _ action.Action = auditAction{}

type controlledMatcher struct {
	result appdecision.MatchResult
}

func (m controlledMatcher) Match(
	_ context.Context,
	_ string,
	_ []apppresenter.IntentDefinition,
) (appdecision.MatchResult, error) {
	return m.result, nil
}
