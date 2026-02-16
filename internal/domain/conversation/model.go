package conversation

import "github.com/google/uuid"

type Conversation struct {
	ID      uuid.UUID
	Channel Channel
	ChatID  int64
	State   State
}

type Channel string

const (
	ChannelTelegram Channel = "telegram"
)

type State string

const (
	StateNew        State = "new"
	StateInProgress State = "in_progress"
	StateClosed     State = "closed"
)