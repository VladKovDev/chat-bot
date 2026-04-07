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
	ChannelWeb Channel = "web"
)