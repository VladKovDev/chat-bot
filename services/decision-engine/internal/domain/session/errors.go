package session

import "errors"

var (
	ErrNotFound = errors.New("session not found")
	ErrInvalidTransition = errors.New("invalid state transition")
)
