package operator

import (
	"time"

	"github.com/google/uuid"
)

type QueueStatus string

const (
	QueueStatusWaiting  QueueStatus = "waiting"
	QueueStatusAccepted QueueStatus = "accepted"
	QueueStatusClosed   QueueStatus = "closed"
)

type Reason string

const (
	ReasonManualRequest         Reason = "manual_request"
	ReasonLowConfidenceRepeated Reason = "low_confidence_repeated"
	ReasonComplaint             Reason = "complaint"
	ReasonBusinessError         Reason = "business_error"
)

type EventType string

const (
	EventQueued   EventType = "queued"
	EventAccepted EventType = "accepted"
	EventClosed   EventType = "closed"
)

type ActorType string

const (
	ActorUser     ActorType = "user"
	ActorOperator ActorType = "operator"
	ActorSystem   ActorType = "system"
)

type QueueItem struct {
	ID                 uuid.UUID
	SessionID          uuid.UUID
	UserID             uuid.UUID
	Status             QueueStatus
	Reason             Reason
	Priority           int
	AssignedOperatorID string
	ContextSnapshot    ContextSnapshot
	CreatedAt          time.Time
	UpdatedAt          time.Time
	AcceptedAt         *time.Time
	ClosedAt           *time.Time
}

type Account struct {
	OperatorID  string
	FixtureID   string
	DisplayName string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Assignment struct {
	ID         uuid.UUID
	QueueID    uuid.UUID
	OperatorID string
	Status     QueueStatus
	AssignedAt time.Time
	ReleasedAt *time.Time
}

type Event struct {
	ID        uuid.UUID
	QueueID   uuid.UUID
	SessionID uuid.UUID
	EventType EventType
	ActorType ActorType
	ActorID   string
	Payload   map[string]interface{}
	CreatedAt time.Time
}

type ContextSnapshot struct {
	LastMessages    []MessageSnapshot `json:"last_messages"`
	ActiveTopic     string            `json:"active_topic"`
	LastIntent      string            `json:"last_intent"`
	Confidence      *float64          `json:"confidence,omitempty"`
	FallbackCount   int               `json:"fallback_count"`
	ActionSummaries []ActionSummary   `json:"action_summaries"`
}

type MessageSnapshot struct {
	SenderType string    `json:"sender_type"`
	Text       string    `json:"text"`
	Intent     string    `json:"intent,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type ActionSummary struct {
	ActionType string    `json:"action_type"`
	Status     string    `json:"status"`
	Summary    string    `json:"summary,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type QueueRequest struct {
	ID              uuid.UUID
	SessionID       uuid.UUID
	UserID          uuid.UUID
	Reason          Reason
	Priority        int
	ContextSnapshot ContextSnapshot
}

type AcceptRequest struct {
	QueueID    uuid.UUID
	OperatorID string
}

type CloseRequest struct {
	QueueID    uuid.UUID
	OperatorID string
}
