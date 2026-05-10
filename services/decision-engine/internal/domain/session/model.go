package session

import (
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
)

// Session represents a user conversation session
type Session struct {
	ID             uuid.UUID
	ChatID         int64
	UserID         uuid.UUID
	Channel        string
	ExternalUserID string
	ClientID       string
	State          state.State
	ActiveTopic    string
	Summary        *string // Optional summary of the conversation
	Version        int
	Status         Status // active, closed
	Metadata       map[string]interface{}
}

// Status represents the session status
type Status string

const (
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

const (
	ChannelWebsite = "website"
	ChannelDevCLI  = "dev-cli"
)

type Identity struct {
	Channel        string
	ExternalUserID string
	ClientID       string
}
