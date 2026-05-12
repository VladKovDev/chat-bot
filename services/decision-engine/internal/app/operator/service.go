package operator

import (
	"context"
	"errors"
	"strings"

	operatorDomain "github.com/VladKovDev/chat-bot/internal/domain/operator"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/google/uuid"
)

type SessionReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (session.Session, error)
}

type Service struct {
	queue    operatorDomain.Repository
	sessions SessionReader
}

func NewService(
	queue operatorDomain.Repository,
	sessions SessionReader,
) *Service {
	return &Service{
		queue:    queue,
		sessions: sessions,
	}
}

func (s *Service) Queue(
	ctx context.Context,
	sessionID uuid.UUID,
	reason operatorDomain.Reason,
	snapshot operatorDomain.ContextSnapshot,
) (operatorDomain.QueueItem, error) {
	return s.QueueWithDecision(ctx, sessionID, reason, snapshot, session.ContextDecision{})
}

func (s *Service) QueueWithDecision(
	ctx context.Context,
	sessionID uuid.UUID,
	reason operatorDomain.Reason,
	snapshot operatorDomain.ContextSnapshot,
	decision session.ContextDecision,
) (operatorDomain.QueueItem, error) {
	reason = operatorDomain.NormalizeReason(reason)
	if err := operatorDomain.ValidateReason(reason); err != nil {
		return operatorDomain.QueueItem{}, err
	}

	sess, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}

	existing, err := s.queue.GetOpenBySession(ctx, sessionID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, operatorDomain.ErrNotFound) {
		return operatorDomain.QueueItem{}, err
	}

	queueID := uuid.New()
	decision.Event = session.EventRequestOperator
	if decision.Metadata == nil {
		decision.Metadata = map[string]interface{}{}
	}
	decision.Metadata["handoff_id"] = queueID.String()
	decision.Metadata["handoff_reason"] = string(reason)

	sessionUpdate, transition, err := session.PrepareContextUpdate(&sess, decision)
	if err != nil {
		return operatorDomain.QueueItem{}, mapSessionTransitionError(err)
	}

	return s.queue.Queue(ctx, operatorDomain.QueueRequest{
		ID:              queueID,
		SessionID:       sessionID,
		UserID:          sess.UserID,
		Reason:          reason,
		ContextSnapshot: snapshot,
	}, sessionUpdate, transition)
}

func (s *Service) Accept(
	ctx context.Context,
	queueID uuid.UUID,
	operatorID string,
) (operatorDomain.QueueItem, error) {
	operatorID = strings.TrimSpace(operatorID)
	if operatorID == "" {
		return operatorDomain.QueueItem{}, operatorDomain.ErrInvalidOperator
	}

	item, err := s.queue.GetByID(ctx, queueID)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}
	sess, err := s.sessions.GetByID(ctx, item.SessionID)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}
	sessionUpdate, transition, err := session.PrepareContextUpdate(&sess, session.ContextDecision{
		Event: session.EventOperatorConnected,
		Metadata: map[string]interface{}{
			"handoff_id":  item.ID.String(),
			"operator_id": operatorID,
		},
	})
	if err != nil {
		return operatorDomain.QueueItem{}, mapSessionTransitionError(err)
	}

	return s.queue.Accept(ctx, operatorDomain.AcceptRequest{
		QueueID:    queueID,
		OperatorID: operatorID,
	}, sessionUpdate, transition)
}

func (s *Service) Close(
	ctx context.Context,
	queueID uuid.UUID,
	operatorID string,
) (operatorDomain.QueueItem, error) {
	operatorID = strings.TrimSpace(operatorID)

	item, err := s.queue.GetByID(ctx, queueID)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}
	sess, err := s.sessions.GetByID(ctx, item.SessionID)
	if err != nil {
		return operatorDomain.QueueItem{}, err
	}
	metadata := map[string]interface{}{
		"handoff_id": item.ID.String(),
	}
	if operatorID != "" {
		metadata["operator_id"] = operatorID
	}
	sessionUpdate, transition, err := session.PrepareContextUpdate(&sess, session.ContextDecision{
		Event:    session.EventOperatorClosed,
		Metadata: metadata,
	})
	if err != nil {
		return operatorDomain.QueueItem{}, mapSessionTransitionError(err)
	}

	return s.queue.Close(ctx, operatorDomain.CloseRequest{
		QueueID:    queueID,
		OperatorID: operatorID,
	}, sessionUpdate, transition)
}

func (s *Service) GetByID(ctx context.Context, queueID uuid.UUID) (operatorDomain.QueueItem, error) {
	return s.queue.GetByID(ctx, queueID)
}

func (s *Service) ListByStatus(
	ctx context.Context,
	status operatorDomain.QueueStatus,
	limit int32,
	offset int32,
) ([]operatorDomain.QueueItem, error) {
	status = operatorDomain.NormalizeStatus(status)
	if err := operatorDomain.ValidateStatus(status); err != nil {
		return nil, err
	}
	return s.queue.ListByStatus(ctx, status, limit, offset)
}

func mapSessionTransitionError(err error) error {
	if errors.Is(err, session.ErrInvalidTransition) {
		return operatorDomain.ErrInvalidTransition
	}
	return err
}
