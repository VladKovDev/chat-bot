package conversation

import (
	"github.com/google/uuid"
)

type Conversation struct {
	ID       uuid.UUID
	ChatID   int64
	State    State
	Metadata map[string]interface{}
}
