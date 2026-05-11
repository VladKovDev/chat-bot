package worker

import (
	"context"
	"time"

	appdecision "github.com/VladKovDev/chat-bot/internal/app/decision"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/message"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

type MessagePersistence interface {
	WithinMessageTransaction(ctx context.Context, fn func(context.Context, MessageTransaction) error) error
}

type MessageTransaction interface {
	CreateMessage(ctx context.Context, msg message.Message) (message.Message, error)
	GetLastMessagesBySessionID(ctx context.Context, sessionID uuid.UUID, limit int32) ([]message.Message, error)
	LogDecision(ctx context.Context, entry DecisionLog) error
	LogAction(ctx context.Context, entry action.Log) error
	ApplyContextDecision(ctx context.Context, sess *session.Session, decision session.ContextDecision) (session.Session, error)
}

type DecisionLog struct {
	ID             uuid.UUID
	SessionID      uuid.UUID
	MessageID      uuid.UUID
	Intent         string
	State          state.State
	ResponseKey    string
	Confidence     *float64
	LowConfidence  bool
	FallbackReason string
	Threshold      *float64
	Candidates     []appdecision.Candidate
	CreatedAt      time.Time
}
