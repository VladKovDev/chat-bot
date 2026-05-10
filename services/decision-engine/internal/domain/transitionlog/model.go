package transitionlog

import (
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/google/uuid"
	"time"
)

type TransitionLog struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	FromMode  session.Mode
	ToMode    session.Mode
	Event     session.Event
	Reason    string
	CreatedAt time.Time
}
