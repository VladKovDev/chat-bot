package session

import (
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/google/uuid"
	"time"
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
	Mode           Mode
	ActiveTopic    string
	LastIntent     string
	FallbackCount  int
	OperatorStatus OperatorStatus
	Summary        *string // Optional summary of the conversation
	Version        int
	Status         Status // active, closed
	Metadata       map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Status represents the session status
type Status string

const (
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

// Mode is the limited BRD conversation mode FSM. Domain topics stay in ActiveTopic.
type Mode string

const (
	ModeStandard          Mode = "standard"
	ModeWaitingOperator   Mode = "waiting_operator"
	ModeOperatorConnected Mode = "operator_connected"
	ModeClosed            Mode = "closed"
)

// OperatorStatus tracks handoff lifecycle without mixing it into topics.
type OperatorStatus string

const (
	OperatorStatusNone      OperatorStatus = "none"
	OperatorStatusWaiting   OperatorStatus = "waiting"
	OperatorStatusConnected OperatorStatus = "connected"
	OperatorStatusClosed    OperatorStatus = "closed"
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

type ContextDecision struct {
	Intent        string
	Topic         string
	LowConfidence bool
	Event         Event
	Metadata      map[string]interface{}
}

type ModeTransition struct {
	SessionID uuid.UUID
	From      Mode
	To        Mode
	Event     Event
	Reason    string
}
