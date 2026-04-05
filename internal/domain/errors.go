package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for domain-level failures.
var (
	ErrEndpointNotFound      = errors.New("endpoint not found")
	ErrEndpointUnreachable   = errors.New("endpoint unreachable")
	ErrProfileNotAvailable   = errors.New("profile not available on endpoint")
	ErrDaemonAlreadyRunning  = errors.New("daemon already running")
	ErrDaemonNotRunning      = errors.New("daemon not running")
	ErrAIProviderUnavailable = errors.New("AI provider unavailable")
)

// RetryExhaustedError is returned when all retry attempts for an endpoint fail.
type RetryExhaustedError struct {
	Endpoint string
	Attempts int
	Last     error
}

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("endpoint %s: all %d attempts failed: %v", e.Endpoint, e.Attempts, e.Last)
}

func (e *RetryExhaustedError) Unwrap() error { return e.Last }
