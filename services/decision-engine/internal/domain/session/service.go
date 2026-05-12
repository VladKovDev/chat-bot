package session

import (
	"context"
	"strings"

	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

type StartResult struct {
	Session Session
	Resumed bool
}

func NormalizeIdentity(identity Identity) Identity {
	return Identity{
		Channel:        strings.TrimSpace(identity.Channel),
		ExternalUserID: strings.TrimSpace(identity.ExternalUserID),
		ClientID:       strings.TrimSpace(identity.ClientID),
	}
}

func ValidateIdentity(identity Identity) error {
	identity = NormalizeIdentity(identity)
	if identity.Channel == "" || (identity.ExternalUserID == "" && identity.ClientID == "") {
		return ErrInvalidIdentity
	}
	return nil
}

func (s *Service) StartSession(ctx context.Context, identity Identity) (StartResult, error) {
	identity = NormalizeIdentity(identity)
	if err := ValidateIdentity(identity); err != nil {
		return StartResult{}, err
	}

	existing, err := s.repo.GetActiveByIdentity(ctx, identity)
	if err == nil {
		normalizeContext(&existing)
		return StartResult{Session: existing, Resumed: true}, nil
	}
	if err != ErrNotFound {
		return StartResult{}, err
	}

	newSession := Session{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Channel:        identity.Channel,
		ExternalUserID: identity.ExternalUserID,
		ClientID:       identity.ClientID,
		State:          state.StateNew,
		Mode:           ModeStandard,
		OperatorStatus: OperatorStatusNone,
		Status:         StatusActive,
		Metadata:       make(map[string]interface{}),
	}
	createdSession, err := s.repo.Create(ctx, newSession)
	if err != nil {
		existing, getErr := s.repo.GetActiveByIdentity(ctx, identity)
		if getErr == nil {
			normalizeContext(&existing)
			return StartResult{Session: existing, Resumed: true}, nil
		}
		return StartResult{}, err
	}

	return StartResult{Session: createdSession, Resumed: false}, nil
}

func (s *Service) LoadSessionByID(ctx context.Context, sessionID uuid.UUID, identity Identity) (*Session, error) {
	identity = NormalizeIdentity(identity)
	if err := ValidateIdentity(identity); err != nil {
		return nil, err
	}

	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status != StatusActive || sess.Channel != identity.Channel {
		return nil, ErrNotFound
	}
	if identity.ExternalUserID != "" {
		if sess.ExternalUserID != identity.ExternalUserID {
			return nil, ErrNotFound
		}
		normalizeContext(&sess)
		return &sess, nil
	}
	if sess.ClientID != identity.ClientID {
		return nil, ErrNotFound
	}
	normalizeContext(&sess)
	return &sess, nil
}

func (s *Service) UpdateSessionState(ctx context.Context, session *Session) (Session, error) {
	normalizeContext(session)
	return s.repo.Update(ctx, *session)
}

func (s *Service) ApplyContextDecision(ctx context.Context, sess *Session, decision ContextDecision) (Session, error) {
	next, transition, err := PrepareContextUpdate(sess, decision)
	if err != nil {
		return Session{}, err
	}
	updated, err := s.repo.UpdateContext(ctx, next, transition)
	if err != nil {
		return Session{}, err
	}
	*sess = updated
	return updated, nil
}

func PrepareContextUpdate(sess *Session, decision ContextDecision) (Session, *ModeTransition, error) {
	if sess == nil {
		return Session{}, nil, ErrNotFound
	}

	next := *sess
	normalizeContext(&next)
	fromMode := next.Mode

	if decision.Event != "" && decision.Event != EventUnknown && decision.Event != EventMessageReceived {
		mode, err := nextModeForEvent(next.Mode, decision.Event)
		if err != nil {
			return Session{}, nil, err
		}
		next.Mode = mode
	}

	topicSwitched := decision.Topic != "" && next.ActiveTopic != "" && next.ActiveTopic != decision.Topic
	if decision.Topic != "" {
		next.ActiveTopic = decision.Topic
	}

	if topicSwitched {
		next.FallbackCount = 0
	}
	if decision.LowConfidence {
		next.FallbackCount++
	} else if decision.Intent != "" {
		next.FallbackCount = 0
	}

	if decision.Intent != "" {
		next.LastIntent = decision.Intent
	} else if topicSwitched {
		next.LastIntent = ""
	}

	next.OperatorStatus = operatorStatusForMode(next.Mode)
	if next.Mode == ModeClosed {
		next.Status = StatusClosed
	} else {
		next.Status = StatusActive
	}
	if next.Metadata == nil {
		next.Metadata = make(map[string]interface{})
	}
	for key, value := range decision.Metadata {
		next.Metadata[key] = value
	}

	var transition *ModeTransition
	if fromMode != next.Mode {
		transition = &ModeTransition{
			SessionID: next.ID,
			From:      fromMode,
			To:        next.Mode,
			Event:     decision.Event,
			Reason:    "context_decision",
		}
	}

	return next, transition, nil
}

func (s *Service) CloseSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.repo.UpdateStatus(ctx, sessionID, StatusClosed)
	return err
}

func normalizeContext(sess *Session) {
	if sess.Mode == "" {
		sess.Mode = ModeStandard
	}
	if sess.OperatorStatus == "" {
		sess.OperatorStatus = operatorStatusForMode(sess.Mode)
	}
	if sess.Metadata == nil {
		sess.Metadata = make(map[string]interface{})
	}
}

func nextModeForEvent(current Mode, event Event) (Mode, error) {
	switch event {
	case EventRequestOperator:
		if current == ModeStandard {
			return ModeWaitingOperator, nil
		}
		if current == ModeWaitingOperator || current == ModeOperatorConnected {
			return current, nil
		}
	case EventOperatorConnected:
		if current == ModeWaitingOperator {
			return ModeOperatorConnected, nil
		}
	case EventOperatorClosed:
		if current == ModeWaitingOperator || current == ModeOperatorConnected {
			return ModeStandard, nil
		}
	case EventResetConversation:
		return ModeStandard, nil
	}

	if event == EventGreeting || event == EventCategorySelected || event == EventResolved ||
		event == EventNotResolved || event == EventConfirmation || event == EventNegation ||
		event == EventGratitude || event == EventClarification {
		return current, nil
	}

	return current, ErrInvalidTransition
}

func operatorStatusForMode(mode Mode) OperatorStatus {
	switch mode {
	case ModeWaitingOperator:
		return OperatorStatusWaiting
	case ModeOperatorConnected:
		return OperatorStatusConnected
	case ModeClosed:
		return OperatorStatusClosed
	default:
		return OperatorStatusNone
	}
}
