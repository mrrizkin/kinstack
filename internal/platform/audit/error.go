package audit

import "errors"

// ErrInvalidEvent indicates an invalid event or query limit.
var ErrInvalidEvent = errors.New("invalid audit event")
