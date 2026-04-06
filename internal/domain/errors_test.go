package domain_test

import (
	"errors"
	"testing"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		domain.ErrEndpointNotFound,
		domain.ErrEndpointUnreachable,
		domain.ErrProfileNotAvailable,
		domain.ErrDaemonAlreadyRunning,
		domain.ErrDaemonNotRunning,
		domain.ErrAIProviderUnavailable,
	}
	for _, err := range sentinels {
		assert.NotEmpty(t, err.Error())
	}
}

func TestRetryExhaustedError_Message(t *testing.T) {
	inner := errors.New("connection refused")
	err := &domain.RetryExhaustedError{
		Endpoint: "http://localhost:6060",
		Attempts: 3,
		Last:     inner,
	}

	assert.Contains(t, err.Error(), "http://localhost:6060")
	assert.Contains(t, err.Error(), "3")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestRetryExhaustedError_Unwrap(t *testing.T) {
	inner := errors.New("timeout")
	err := &domain.RetryExhaustedError{
		Endpoint: "app",
		Attempts: 3,
		Last:     inner,
	}

	assert.True(t, errors.Is(err, inner))
}
