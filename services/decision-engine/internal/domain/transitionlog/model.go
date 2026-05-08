package transitionlog

import (
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
	"time"
)

type TransitionLog struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	FromState state.State
	ToState   state.State
	CreatedAt time.Time
}
