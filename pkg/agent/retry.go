// Package agent provides the core agent loop functionality with retry, parallel execution, and auto mode support.
package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries int           // Maximum number of retry attempts (default: 3)
	Delay      time.Duration // Fixed delay between retries (default: 1s)
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		Delay:      time.Second,
	}
}

// RetryableError is an interface for errors that can be retried
type RetryableError interface {
	error
	IsRetryable() bool
}

// RateLimitError indicates rate limiting was encountered
type RateLimitError struct {
	Err        error
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string { return fmt.Sprintf("rate limited: %v", e.Err) }
func (e *RateLimitError) IsRetryable() bool { return true }

// AuthenticationError indicates an authentication failure
type AuthenticationError struct {
	Err error
}

func (e *AuthenticationError) Error() string { return fmt.Sprintf("authentication failed: %v", e.Err) }
func (e *AuthenticationError) IsRetryable() bool { return false }

// TimeoutError indicates a timeout occurred
type TimeoutError struct {
	Err error
}

func (e *TimeoutError) Error() string { return fmt.Sprintf("timeout: %v", e.Err) }
func (e *TimeoutError) IsRetryable() bool { return true }

// TransientError indicates a transient error that can be retried
type TransientError struct {
	Err error
}

func (e *TransientError) Error() string { return fmt.Sprintf("transient error: %v", e.Err) }
func (e *TransientError) IsRetryable() bool { return true }

// ClassifyError classifies an error and returns an appropriate RetryableError
func ClassifyError(err error) RetryableError {
	if err == nil {
		return nil
	}

	// Check if already a RetryableError
	var re RetryableError
	if errors.As(err, &re) {
		return re
	}

	errStr := err.Error()

	// Check for common error patterns
	if contains(errStr, []string{"rate limit", "rate_limit", "429", "too many requests"}) {
		return &RateLimitError{Err: err, RetryAfter: 0}
	}

	if contains(errStr, []string{"authentication", "auth", "401", "unauthorized", "api key", "invalid key"}) {
		return &AuthenticationError{Err: err}
	}

	if contains(errStr, []string{"timeout", "timed out", "deadline", "context deadline"}) {
		return &TimeoutError{Err: err}
	}

	// Default to transient (could be retried)
	return &TransientError{Err: err}
}

func contains(s string, substrs []string) bool {
	s = strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// RetryOptions configures the retry behavior
type RetryOptions struct {
	MaxRetries int
	Delay      time.Duration
	// IsRetryableFn is called to determine if an error should be retried
	// If nil, defaults to checking if error is a RetryableError
	IsRetryableFn func(error) bool
}

// WithRetry executes a function with retry logic
func WithRetry[T any](ctx context.Context, fn func() (T, error), opts RetryOptions) (T, error) {
	var lastErr error

	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	delay := opts.Delay
	if delay <= 0 {
		delay = time.Second
	}

	isRetryable := opts.IsRetryableFn
	if isRetryable == nil {
		isRetryable = func(err error) bool {
			if re, ok := err.(RetryableError); ok {
				return re.IsRetryable()
			}
			// Default: retry unless it's an authentication error
			var authErr *AuthenticationError
			return !errors.As(err, &authErr)
		}
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			// Success
			if attempt > 0 {
				logRetrySuccess(attempt)
			}
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if attempt < maxRetries && isRetryable(err) {
			logRetryAttempt(attempt+1, maxRetries, err)

			// Check for rate limit with specific retry-after
			var rateLimitErr *RateLimitError
			if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
				delay = rateLimitErr.RetryAfter
			}

			select {
			case <-ctx.Done():
				return *new(T), ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		// No more retries or error is not retryable
		break
	}

	return *new(T), lastErr
}

// WithRetryVoid executes a void function with retry logic
func WithRetryVoid(ctx context.Context, fn func() error, opts RetryOptions) error {
	_, err := WithRetry(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	}, opts)
	return err
}

// logRetryAttempt logs a retry attempt
func logRetryAttempt(attempt, max int, err error) {
	logger.DebugCF("agent", "Retry attempt",
		map[string]interface{}{
			"attempt": attempt,
			"max":     max,
			"error":   err.Error(),
		})
}

// logRetrySuccess logs successful retry after failure
func logRetrySuccess(attempt int) {
	logger.InfoCF("agent", "Retry succeeded",
		map[string]interface{}{
			"attempt": attempt,
		})
}
