package conversation

import "errors"

var (
	ErrNotFound = errors.New("conversation not found")
	ErrInvalidTransition = errors.New("invalid state transition")
)
