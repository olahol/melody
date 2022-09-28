package melody

import "errors"

var (
	ErrClosed            = errors.New("melody instance is closed")
	ErrSessionClosed     = errors.New("session is closed")
	ErrWriteClosed       = errors.New("tried to write to closed a session")
	ErrConnNil           = errors.New("connection was not set")
	ErrMessageBufferFull = errors.New("session message buffer is full")
)
