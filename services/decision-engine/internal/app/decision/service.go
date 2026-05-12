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
	ContextMatchThreshold   = 0.70
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
	FallbackReason          string
	Event                   session.Event
	UseActionResponseSelect bool
}

type Service struct {
	intentsByKey map[string]apppresenter.IntentDefinition
	intents      []apppresenter.IntentDefinition
	matcher      Matcher
	knowledge    KnowledgeSearcher
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

func (s *Service) SetKnowledgeSearcher(searcher KnowledgeSearcher) {
	s.knowledge = searcher
}

func (s *Service) Decide(
	ctx context.Context,
	sess session.Session,
	_ []message.Message,
	text string,
) (Result, error) {
	if contextual, ok := s.resolveIdentifierFollowUp(sess, text); ok {
		return contextual, nil
	}

	match, err := s.matcher.Match(ctx, text, s.intents)
	if err != nil {
		return Result{}, fmt.Errorf("match intent: %w", err)
	}

	if scoped, ok, scopedErr := s.topicScopedMatch(ctx, sess, text, match); scopedErr != nil {
		return Result{}, fmt.Errorf("match scoped intent: %w", scopedErr)
	} else if ok {
		match = scoped
	}

	match = applyMixedDomainAmbiguityPolicy(text, match)

	if contextual, ok := s.resolveContextualFollowUp(sess, text, match); ok {
		return contextual, nil
	}
	if pending, ok := s.resolvePendingBusinessLookupFallback(sess, text, match); ok {
		return pending, nil
	}

	confidence := match.Confidence
	if match.IntentKey == "" {
		return s.lowConfidenceResult(sess, confidence, match.Candidates, match.FallbackReason), nil
	}

	intentDefinition, ok := s.intentsByKey[match.IntentKey]
	if !ok {
		return s.lowConfidenceResult(sess, confidence, match.Candidates, match.FallbackReason), nil
	}

	if s.isLowConfidence(match) {
		if promoted, ok := s.promoteContextualLowConfidence(sess, match, intentDefinition, text); ok {
			return promoted, nil
		}
		return s.lowConfidenceResult(sess, confidence, match.Candidates, match.FallbackReason), nil
	}

	candidates := append([]Candidate(nil), match.Candidates...)
	if enriched, err := s.retrieveKnowledgeCandidate(ctx, text, intentDefinition); err != nil {
		return Result{}, fmt.Errorf("retrieve knowledge candidate: %w", err)
	} else if enriched != nil {
		candidates = append(candidates, *enriched)
	}

	return resultForIntent(sess, intentDefinition, text, confidencePtr(confidence), candidates), nil
}

func (s *Service) topicScopedMatch(
	ctx context.Context,
	sess session.Session,
	text string,
	global MatchResult,
) (MatchResult, bool, error) {
	activeTopic := strings.TrimSpace(sess.ActiveTopic)
	if activeTopic == "" {
		return MatchResult{}, false, nil
	}
	if !s.isLowConfidence(global) && topCandidateCategory(global) == activeTopic {
		return MatchResult{}, false, nil
	}

	scopedIntents := s.intentsForTopic(activeTopic)
	if len(scopedIntents) == 0 {
		return MatchResult{}, false, nil
	}

	scoped, err := s.matcher.Match(ctx, text, scopedIntents)
	if err != nil {
		return MatchResult{}, false, err
	}
	if scoped.IntentKey == "" {
		return MatchResult{}, false, nil
	}
	if topCandidateCategory(scoped) != "" && topCandidateCategory(scoped) != activeTopic {
		return MatchResult{}, false, nil
	}
	if !s.isLowConfidence(scoped) || topCandidateCategory(scoped) == activeTopic {
		return scoped, true, nil
	}

	return MatchResult{}, false, nil
}

func (s *Service) resolveIdentifierFollowUp(sess session.Session, text string) (Result, bool) {
	intentDefinition, ok := s.pendingBusinessLookupIntent(sess)
	if !ok {
		return Result{}, false
	}

	identifier, _ := extractIdentifier(text, intentDefinition.Action)
	if identifier == "" {
		return Result{}, false
	}

	return resultForIntent(sess, intentDefinition, text, confidencePtr(1), nil), true
}

func (s *Service) retrieveKnowledgeCandidate(
	ctx context.Context,
	text string,
	intentDefinition apppresenter.IntentDefinition,
) (*Candidate, error) {
	if s.knowledge == nil {
		return nil, nil
	}
	return s.knowledge.Retrieve(ctx, text, intentDefinition)
}

func (s *Service) intentsForTopic(topic string) []apppresenter.IntentDefinition {
	scoped := make([]apppresenter.IntentDefinition, 0)
	for _, intentDefinition := range s.intents {
		switch intentDefinition.Category {
		case topic, "system", "operator":
			scoped = append(scoped, intentDefinition)
		}
	}
	return scoped
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
			return s.lowConfidenceResult(sess, 0, nil, defaultLowConfidence), nil
		}
		return resultForIntent(
			sess,
			intentDefinition,
			text,
			confidencePtr(1),
			[]Candidate{deterministicQuickReplyCandidate(selection, intentKey, text)},
		), nil
	case "request_operator":
		intentDefinition, ok := s.intentsByKey["request_operator"]
		if !ok {
			return Result{
				Intent:      "request_operator",
				State:       state.StateEscalatedToOperator,
				Topic:       sess.ActiveTopic,
				ResponseKey: "operator_handoff_requested",
				Confidence:  confidencePtr(1),
				Candidates: []Candidate{
					deterministicQuickReplyCandidate(selection, "request_operator", text),
				},
				Event:   session.EventRequestOperator,
				Actions: []string{action.ActionEscalateToOperator},
				ActionContext: map[string]any{
					"handoff_reason": "manual_request",
				},
			}, nil
		}
		return resultForIntent(
			sess,
			intentDefinition,
			text,
			confidencePtr(1),
			[]Candidate{deterministicQuickReplyCandidate(selection, "request_operator", text)},
		), nil
	case "send_text":
		if strings.TrimSpace(text) == "" {
			return Result{}, fmt.Errorf("quick reply %q send_text payload.text is required", selection.ID)
		}
		return s.Decide(ctx, sess, history, text)
	default:
		return Result{}, fmt.Errorf("quick reply %q action %q is unsupported", selection.ID, selection.Action)
	}
}

func deterministicQuickReplyCandidate(selection QuickReplySelection, intentKey string, text string) Candidate {
	candidateText := strings.TrimSpace(text)
	if candidateText == "" {
		candidateText = firstNonEmpty(
			quickReplyPayloadString(selection.Payload, "text"),
			quickReplyPayloadString(selection.Payload, "intent"),
			strings.TrimSpace(selection.ID),
			strings.TrimSpace(selection.Action),
		)
	}
	return Candidate{
		IntentKey:  intentKey,
		Confidence: 1,
		Source:     CandidateSourceQuickReplyIntent,
		Text:       candidateText,
		Metadata: map[string]any{
			"quick_reply_id":     strings.TrimSpace(selection.ID),
			"quick_reply_action": strings.TrimSpace(selection.Action),
		},
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

func (s *Service) resolveContextualFollowUp(sess session.Session, text string, match MatchResult) (Result, bool) {
	if !s.isLowConfidence(match) {
		return Result{}, false
	}

	normalized := normalizeText(text)
	if normalized == "" {
		return Result{}, false
	}

	if isNegativeFollowUp(normalized) {
		if result, ok := s.resultForIntentKey(sess, "return_to_menu", text, 1); ok {
			return result, true
		}
	}

	if isAffirmativeFollowUp(normalized) {
		if intentDefinition, ok := s.pendingBusinessLookupIntent(sess); ok {
			return resultForIntent(sess, intentDefinition, text, confidencePtr(1), nil), true
		}
	}

	activeTopic := strings.TrimSpace(sess.ActiveTopic)
	if activeTopic == "" || !isShortContextualFollowUp(normalized) {
		return Result{}, false
	}

	intentKey := contextualIntentForTopic(activeTopic, normalized)
	if intentKey == "" {
		return Result{}, false
	}

	result, ok := s.resultForIntentKey(sess, intentKey, text, ContextMatchThreshold)
	if !ok {
		return Result{}, false
	}
	return result, true
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

func (s *Service) lowConfidenceResult(sess session.Session, confidence float64, candidates []Candidate, fallbackReason string) Result {
	if sess.FallbackCount >= 1 {
		return Result{
			Intent:         "unknown",
			State:          state.StateEscalatedToOperator,
			ResponseKey:    "operator_handoff_requested",
			Confidence:     confidencePtr(confidence),
			Candidates:     append([]Candidate(nil), candidates...),
			LowConfidence:  true,
			FallbackReason: "low_confidence_repeated",
			Event:          session.EventRequestOperator,
			Actions:        []string{action.ActionEscalateToOperator},
			ActionContext: map[string]any{
				"handoff_reason": "low_confidence_repeated",
			},
		}
	}

	result := Result{
		Intent:         "unknown",
		State:          state.StateWaitingClarification,
		ResponseKey:    "clarify_request",
		Confidence:     confidencePtr(confidence),
		Candidates:     append([]Candidate(nil), candidates...),
		LowConfidence:  true,
		FallbackReason: firstNonEmpty(fallbackReason, defaultLowConfidence),
		Event:          session.EventMessageReceived,
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

type mixedDomainAmbiguityRule struct {
	categories             [2]string
	domainSignals          map[string][]string
	sharedSignals          []string
	disambiguatingSignals  map[string][]string
	minSecondaryConfidence float64
}

var mixedDomainAmbiguityRules = []mixedDomainAmbiguityRule{
	{
		categories: [2]string{"workspace", "services"},
		domainSignals: map[string][]string{
			"workspace": {"место", "места", "коворкинг", "стол", "офис"},
			"services":  {"услуга", "услуги", "процедур", "стриж", "маник", "педик", "массаж", "макияж", "окраш"},
		},
	},
	{
		categories:    [2]string{"account", "tech_issue"},
		sharedSignals: []string{"код"},
		disambiguatingSignals: map[string][]string{
			"account":    {"аккаунт", "профиль", "кабинет", "почт"},
			"tech_issue": {"сайт", "логин", "войти", "авториза", "смс", "браузер", "телефон"},
		},
		minSecondaryConfidence: 0.85,
	},
}

func applyMixedDomainAmbiguityPolicy(text string, match MatchResult) MatchResult {
	if len(match.Candidates) < 2 {
		return match
	}

	normalized := normalizeText(text)
	if normalized == "" {
		return match
	}

	primary := candidateCategory(match.Candidates[0])
	secondary := candidateCategory(match.Candidates[1])
	if primary == "" || secondary == "" || primary == secondary {
		return match
	}

	for _, rule := range mixedDomainAmbiguityRules {
		if !rule.matches(primary, secondary) {
			continue
		}
		if !rule.triggers(normalized, primary, secondary, match.Candidates[1].Confidence) {
			continue
		}
		match.LowConfidence = true
		match.FallbackReason = firstNonEmpty(match.FallbackReason, defaultAmbiguousMatch)
		return match
	}

	return match
}

func (r mixedDomainAmbiguityRule) matches(left, right string) bool {
	return (r.categories[0] == left && r.categories[1] == right) ||
		(r.categories[0] == right && r.categories[1] == left)
}

func (r mixedDomainAmbiguityRule) triggers(normalized, primary, secondary string, secondaryConfidence float64) bool {
	if len(r.domainSignals) > 0 &&
		containsAnySignal(normalized, r.domainSignals[primary]) &&
		containsAnySignal(normalized, r.domainSignals[secondary]) {
		return true
	}

	if r.minSecondaryConfidence > 0 && secondaryConfidence < r.minSecondaryConfidence {
		return false
	}
	if len(r.sharedSignals) == 0 || !containsAnySignal(normalized, r.sharedSignals) {
		return false
	}
	if containsAnySignal(normalized, r.disambiguatingSignals[primary]) {
		return false
	}
	if containsAnySignal(normalized, r.disambiguatingSignals[secondary]) {
		return false
	}
	return true
}

func containsAnySignal(normalized string, signals []string) bool {
	for _, signal := range signals {
		if strings.Contains(normalized, signal) {
			return true
		}
	}
	return false
}

func (s *Service) promoteContextualLowConfidence(
	sess session.Session,
	match MatchResult,
	intentDefinition apppresenter.IntentDefinition,
	text string,
) (Result, bool) {
	if match.FallbackReason != defaultLowConfidence {
		return Result{}, false
	}
	if strings.TrimSpace(sess.ActiveTopic) == "" {
		return Result{}, false
	}
	if match.Confidence < ContextMatchThreshold {
		return Result{}, false
	}
	if topCandidateCategory(match) != strings.TrimSpace(sess.ActiveTopic) {
		return Result{}, false
	}

	return resultForIntent(sess, intentDefinition, text, confidencePtr(match.Confidence), match.Candidates), true
}

func (s *Service) resolvePendingBusinessLookupFallback(sess session.Session, text string, match MatchResult) (Result, bool) {
	if !s.isLowConfidence(match) {
		return Result{}, false
	}
	intentDefinition, ok := s.pendingBusinessLookupIntent(sess)
	if !ok {
		return Result{}, false
	}

	candidates := []Candidate{
		{
			IntentKey:  intentDefinition.Key,
			Confidence: 1,
			Source:     CandidateSourceContextualRule,
			Text:       text,
			Metadata: map[string]any{
				"category": intentDefinition.Category,
				"reason":   "pending_identifier_retry",
			},
		},
	}
	result := resultForIntent(sess, intentDefinition, text, confidencePtr(1), candidates)
	if intentDefinition.Key == "ask_booking_status" {
		result.ResponseKey = "booking_request_identifier_retry"
	}

	return result, true
}

func (s *Service) pendingBusinessLookupIntent(sess session.Session) (apppresenter.IntentDefinition, bool) {
	if sess.State != state.StateWaitingForIdentifier {
		return apppresenter.IntentDefinition{}, false
	}

	lastIntent := strings.TrimSpace(sess.LastIntent)
	if lastIntent == "" {
		return apppresenter.IntentDefinition{}, false
	}

	intentDefinition, ok := s.intentsByKey[lastIntent]
	if !ok || intentDefinition.ResolutionType != "business_lookup" {
		return apppresenter.IntentDefinition{}, false
	}

	return intentDefinition, true
}

func (s *Service) resultForIntentKey(
	sess session.Session,
	intentKey string,
	text string,
	confidence float64,
) (Result, bool) {
	intentDefinition, ok := s.intentsByKey[intentKey]
	if !ok {
		return Result{}, false
	}
	candidates := []Candidate{
		{
			IntentKey:  intentKey,
			Confidence: confidence,
			Source:     CandidateSourceContextualRule,
			Text:       text,
			Metadata: map[string]any{
				"category": intentDefinition.Category,
				"reason":   "contextual_topic_rule",
			},
		},
	}
	return resultForIntent(sess, intentDefinition, text, confidencePtr(confidence), candidates), true
}

type contextualIntentRule struct {
	intentKey string
	signals   []string
}

var contextualTopicRules = map[string][]contextualIntentRule{
	"booking": {
		{intentKey: "ask_booking_status", signals: []string{"статус", "что с записью", "моя запись", "бронь"}},
		{intentKey: "ask_cancellation_rules", signals: []string{"отмен", "не приду"}},
		{intentKey: "ask_reschedule_rules", signals: []string{"перенос", "перенести", "другое время"}},
		{intentKey: "ask_booking_info", signals: []string{"как запис", "запись"}},
	},
	"workspace": {
		{intentKey: "ask_workspace_status", signals: []string{"статус", "бронь", "место забронировано"}},
		{intentKey: "ask_workspace_prices", signals: []string{"сколько стоит", "цена", "прайс", "стоимость", "тариф"}},
		{intentKey: "ask_workspace_rules", signals: []string{"правила", "условия", "можно", "нельзя"}},
		{intentKey: "ask_workspace_info", signals: []string{"коворкинг", "рабочие места", "места"}},
	},
	"payment": {
		{intentKey: "ask_payment_status", signals: []string{"статус", "платеж", "оплат"}},
		{intentKey: "ask_refund_rules", signals: []string{"возврат", "вернуть"}},
	},
	"account": {
		{intentKey: "ask_account_status", signals: []string{"статус", "аккаунт", "профиль"}},
		{intentKey: "account_code_not_received", signals: []string{"код", "подтверждение"}},
		{intentKey: "forgot_password", signals: []string{"пароль", "доступ"}},
		{intentKey: "ask_account_help", signals: []string{"помощь", "кабинет"}},
	},
	"services": {
		{intentKey: "ask_prices", signals: []string{"сколько стоит", "цена", "прайс", "стоимость"}},
		{intentKey: "ask_rules", signals: []string{"правила", "условия"}},
		{intentKey: "ask_location", signals: []string{"адрес", "график", "часы", "где"}},
		{intentKey: "ask_faq", signals: []string{"faq", "частые вопросы"}},
		{intentKey: "show_contacts", signals: []string{"контакты", "телефон", "почта"}},
	},
	"tech_issue": {
		{intentKey: "ask_site_problem", signals: []string{"сайт", "страница"}},
		{intentKey: "login_not_working", signals: []string{"логин", "войти", "авториза"}},
		{intentKey: "code_not_received", signals: []string{"код", "смс"}},
	},
	"complaint": {
		{intentKey: "report_complaint", signals: []string{"жалоб", "оператор", "специалист"}},
	},
}

func contextualIntentForTopic(topic, normalized string) string {
	rules := contextualTopicRules[topic]
	for _, rule := range rules {
		for _, signal := range rule.signals {
			if strings.Contains(normalized, signal) {
				return rule.intentKey
			}
		}
	}
	return ""
}

func isShortContextualFollowUp(normalized string) bool {
	tokens := strings.Fields(normalized)
	return len(tokens) > 0 && len(tokens) <= 5
}

func isAffirmativeFollowUp(normalized string) bool {
	switch normalized {
	case "да", "ага", "угу", "ок", "хорошо":
		return true
	default:
		return false
	}
}

func isNegativeFollowUp(normalized string) bool {
	switch normalized {
	case "нет", "неа", "не хочу", "не нужно":
		return true
	default:
		return false
	}
}

func topCandidateCategory(match MatchResult) string {
	if len(match.Candidates) == 0 {
		return ""
	}
	return candidateCategory(match.Candidates[0])
}

func candidateCategory(candidate Candidate) string {
	if candidate.Metadata == nil {
		return ""
	}
	raw, ok := candidate.Metadata["category"]
	if !ok {
		return ""
	}
	category, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(category)
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
