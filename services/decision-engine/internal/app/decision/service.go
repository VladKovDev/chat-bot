package decision

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

const (
	DefaultMatchThreshold   = 0.78
	DefaultAmbiguityDelta   = 0.08
	defaultLowConfidence    = "low_confidence"
	defaultAmbiguousMatch   = "ambiguous_match"
	defaultNoSemanticIntent = "no_semantic_intent"
)

type Candidate struct {
	IntentID   string         `json:"intent_id,omitempty"`
	IntentKey  string         `json:"intent_key"`
	Confidence float64        `json:"confidence"`
	Source     string         `json:"source,omitempty"`
	Text       string         `json:"text,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type MatchResult struct {
	IntentKey      string
	Confidence     float64
	AmbiguityDelta float64
	LowConfidence  bool
	FallbackReason string
	Candidates     []Candidate
}

type QuickReplySelection struct {
	ID      string
	Action  string
	Payload map[string]any
}

type Matcher interface {
	Match(ctx context.Context, text string, intents []apppresenter.IntentDefinition) (MatchResult, error)
}

type Result struct {
	Intent                  string
	State                   state.State
	Topic                   string
	ResponseKey             string
	Actions                 []string
	ActionContext           map[string]any
	Confidence              *float64
	Candidates              []Candidate
	LowConfidence           bool
	Event                   session.Event
	UseActionResponseSelect bool
}

type Service struct {
	intentsByKey map[string]apppresenter.IntentDefinition
	intents      []apppresenter.IntentDefinition
	matcher      Matcher
	logger       logger.Logger
}

func NewService(
	catalog *apppresenter.IntentCatalog,
	matcher Matcher,
	logger logger.Logger,
) (*Service, error) {
	if catalog == nil || len(catalog.Intents) == 0 {
		return nil, fmt.Errorf("intent catalog is empty")
	}
	if matcher == nil {
		matcher = NewCatalogMatcher()
	}

	intentsByKey := make(map[string]apppresenter.IntentDefinition, len(catalog.Intents))
	intents := make([]apppresenter.IntentDefinition, 0, len(catalog.Intents))
	for _, intentDefinition := range catalog.Intents {
		intentsByKey[intentDefinition.Key] = intentDefinition
		intents = append(intents, intentDefinition)
	}

	return &Service{
		intentsByKey: intentsByKey,
		intents:      intents,
		matcher:      matcher,
		logger:       logger,
	}, nil
}

func (s *Service) Decide(
	ctx context.Context,
	sess session.Session,
	_ []message.Message,
	text string,
) (Result, error) {
	match, err := s.matcher.Match(ctx, text, s.intents)
	if err != nil {
		return Result{}, fmt.Errorf("match intent: %w", err)
	}

	confidence := match.Confidence
	if match.IntentKey == "" || s.isLowConfidence(match) {
		return s.lowConfidenceResult(sess, confidence, match.Candidates), nil
	}

	intentDefinition, ok := s.intentsByKey[match.IntentKey]
	if !ok {
		return s.lowConfidenceResult(sess, confidence, match.Candidates), nil
	}

	return resultForIntent(sess, intentDefinition, text, confidencePtr(confidence), match.Candidates), nil
}

func (s *Service) DecideQuickReply(
	ctx context.Context,
	sess session.Session,
	history []message.Message,
	selection QuickReplySelection,
	text string,
) (Result, error) {
	switch strings.TrimSpace(selection.Action) {
	case "select_intent":
		intentKey := quickReplyPayloadString(selection.Payload, "intent")
		if intentKey == "" {
			return Result{}, fmt.Errorf("quick reply %q select_intent payload.intent is required", selection.ID)
		}
		intentDefinition, ok := s.intentsByKey[intentKey]
		if !ok {
			return s.lowConfidenceResult(sess, 0, nil), nil
		}
		return resultForIntent(sess, intentDefinition, text, confidencePtr(1), nil), nil
	case "request_operator":
		intentDefinition, ok := s.intentsByKey["request_operator"]
		if !ok {
			return Result{
				Intent:      "request_operator",
				State:       state.StateEscalatedToOperator,
				Topic:       sess.ActiveTopic,
				ResponseKey: "operator_handoff_requested",
				Confidence:  confidencePtr(1),
				Event:       session.EventRequestOperator,
				Actions:     []string{action.ActionEscalateToOperator},
				ActionContext: map[string]any{
					"handoff_reason": "manual_request",
				},
			}, nil
		}
		return resultForIntent(sess, intentDefinition, text, confidencePtr(1), nil), nil
	case "send_text":
		if strings.TrimSpace(text) == "" {
			return Result{}, fmt.Errorf("quick reply %q send_text payload.text is required", selection.ID)
		}
		return s.Decide(ctx, sess, history, text)
	default:
		return Result{}, fmt.Errorf("quick reply %q action %q is unsupported", selection.ID, selection.Action)
	}
}

func resultForIntent(
	sess session.Session,
	intentDefinition apppresenter.IntentDefinition,
	text string,
	confidence *float64,
	candidates []Candidate,
) Result {
	result := Result{
		Intent:      intentDefinition.Key,
		State:       baseStateForIntent(intentDefinition),
		Topic:       topicForCategory(intentDefinition.Category),
		ResponseKey: intentDefinition.ResponseKey,
		Confidence:  confidence,
		Candidates:  append([]Candidate(nil), candidates...),
		Event:       session.EventMessageReceived,
	}

	switch intentDefinition.Key {
	case "greeting":
		result.State = state.StateWaitingForCategory
		result.Event = session.EventGreeting
	case "return_to_menu":
		result.State = state.StateWaitingForCategory
	case "reset_conversation":
		result.State = state.StateWaitingForCategory
		result.Event = session.EventResetConversation
	case "goodbye":
		result.State = state.StateClosed
	}

	switch intentDefinition.ResolutionType {
	case "operator_handoff":
		result.State = state.StateEscalatedToOperator
		result.Event = session.EventRequestOperator
		result.ResponseKey = firstNonEmpty(intentDefinition.ResponseKey, "operator_handoff_requested")
		result.Topic = topicForCategory(intentDefinition.Category)
		result.Actions = []string{action.ActionEscalateToOperator}
		result.ActionContext = map[string]any{
			"handoff_reason": handoffReasonForIntent(intentDefinition.Key),
		}
		return result
	case "business_lookup":
		identifier, identifierType := extractIdentifier(text, intentDefinition.Action)
		if identifier == "" {
			result.State = state.StateWaitingForIdentifier
			result.ResponseKey = firstNonEmpty(intentDefinition.FallbackResponseKey, intentDefinition.ResponseKey)
			return result
		}

		result.Actions = []string{intentDefinition.Action}
		result.ActionContext = map[string]any{
			"provided_identifier": identifier,
			"identifier_type":     identifierType,
		}
		result.UseActionResponseSelect = true
		return result
	default:
		result.ResponseKey = firstNonEmpty(intentDefinition.ResponseKey, "clarify_request")
		if intentDefinition.Key == "unknown" {
			result.LowConfidence = true
		}
		return result
	}
}

func quickReplyPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func (s *Service) isLowConfidence(match MatchResult) bool {
	if match.IntentKey == "" {
		return true
	}
	if match.LowConfidence {
		return true
	}
	if match.Confidence < DefaultMatchThreshold {
		return true
	}
	if len(match.Candidates) < 2 {
		return false
	}
	return match.Candidates[0].Confidence-match.Candidates[1].Confidence < DefaultAmbiguityDelta
}

func (s *Service) lowConfidenceResult(sess session.Session, confidence float64, candidates []Candidate) Result {
	result := Result{
		Intent:        "unknown",
		State:         state.StateWaitingClarification,
		ResponseKey:   "clarify_request",
		Confidence:    confidencePtr(confidence),
		Candidates:    append([]Candidate(nil), candidates...),
		LowConfidence: true,
		Event:         session.EventMessageReceived,
	}

	if sess.FallbackCount >= 1 {
		result.State = state.StateEscalatedToOperator
		result.ResponseKey = "operator_handoff_requested"
		result.Event = session.EventRequestOperator
		result.Actions = []string{action.ActionEscalateToOperator}
		result.ActionContext = map[string]any{
			"handoff_reason": "low_confidence_repeated",
		}
	}

	return result
}

func handoffReasonForIntent(intentKey string) string {
	if intentKey == "report_complaint" || strings.HasPrefix(intentKey, "complaint_") {
		return "complaint"
	}
	return "manual_request"
}

func baseStateForIntent(intentDefinition apppresenter.IntentDefinition) state.State {
	switch intentDefinition.Category {
	case "booking":
		return state.StateBooking
	case "workspace":
		return state.StateWorkspace
	case "payment":
		return state.StatePayment
	case "tech_issue":
		return state.StateTechIssue
	case "account":
		return state.StateAccount
	case "services":
		return state.StateServices
	case "complaint":
		return state.StateComplaint
	case "other":
		return state.StateOther
	case "operator":
		return state.StateEscalatedToOperator
	case "fallback":
		return state.StateWaitingClarification
	default:
		return state.StateWaitingForCategory
	}
}

func topicForCategory(category string) string {
	switch category {
	case "booking", "workspace", "payment", "tech_issue", "account", "services", "complaint", "other":
		return category
	default:
		return ""
	}
}

func confidencePtr(value float64) *float64 {
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var (
	bookingIdentifierPattern   = regexp.MustCompile(`(БРГ|BRG)-\d{6}`)
	workspaceIdentifierPattern = regexp.MustCompile(`WS-\d{4}|WRK-(HOT|FIX|OFC1|OFC4)-\d{3}`)
	paymentIdentifierPattern   = regexp.MustCompile(`PAY-[A-Z0-9-]{3,}`)
	userIdentifierPattern      = regexp.MustCompile(`usr-\d{6}`)
	phoneIdentifierPattern     = regexp.MustCompile(`\+7 \(\d{3}\) \d{3}-\d{2}-\d{2}|\b\d{10}\b`)
	emailIdentifierPattern     = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

func extractIdentifier(text, actionName string) (string, string) {
	switch actionName {
	case action.ActionFindBooking:
		if identifier := bookingIdentifierPattern.FindString(text); identifier != "" {
			return identifier, "booking_number"
		}
		if identifier := phoneIdentifierPattern.FindString(text); identifier != "" {
			return identifier, "phone"
		}
	case action.ActionFindWorkspaceBooking:
		if identifier := workspaceIdentifierPattern.FindString(text); identifier != "" {
			return identifier, "workspace_booking"
		}
	case action.ActionFindPayment:
		if identifier := paymentIdentifierPattern.FindString(text); identifier != "" {
			return identifier, "payment_id"
		}
	case action.ActionFindUserAccount:
		if identifier := userIdentifierPattern.FindString(text); identifier != "" {
			return identifier, "user_id"
		}
		if identifier := emailIdentifierPattern.FindString(text); identifier != "" {
			return identifier, "email"
		}
		if identifier := phoneIdentifierPattern.FindString(text); identifier != "" {
			return identifier, "phone"
		}
	}

	return "", ""
}
