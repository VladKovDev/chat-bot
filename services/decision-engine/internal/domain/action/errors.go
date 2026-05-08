package action

import "errors"

var (
	ErrNotFound      = errors.New("action log not found")
	ErrInvalidAction = errors.New("invalid action type")
)
