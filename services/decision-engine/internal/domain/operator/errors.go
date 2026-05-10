package operator

import "errors"

var (
	ErrNotFound          = errors.New("operator handoff not found")
	ErrInvalidReason     = errors.New("invalid operator handoff reason")
	ErrInvalidStatus     = errors.New("invalid operator handoff status")
	ErrInvalidTransition = errors.New("invalid operator handoff transition")
	ErrInvalidOperator   = errors.New("invalid operator id")
)
