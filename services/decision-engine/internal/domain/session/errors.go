package session

import "errors"

var (
	ErrNotFound          = errors.New("session not found")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrInvalidIdentity   = errors.New("session identity requires channel and external_user_id or client_id")
)
