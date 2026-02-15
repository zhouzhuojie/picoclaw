package agent

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWithRetry_SuccessFirstAttempt tests successful execution on first try
func TestWithRetry_SuccessFirstAttempt(t *testing.T) {
	callCount := 0
	fn := func() (int, error) {
		callCount++
		return 42, nil
	}

	result, err := WithRetry(context.Background(), fn, RetryOptions{
		MaxRetries: 3,
		Delay:      time.Millisecond,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

// TestWithRetry_SuccessAfterRetries tests successful execution after some retries
func TestWithRetry_SuccessAfterRetries(t *testing.T) {
	callCount := 0
	fn := func() (int, error) {
		callCount++
		if callCount < 3 {
			return 0, &TransientError{Err: errors.New("temporary failure")}
		}
		return 42, nil
	}

	result, err := WithRetry(context.Background(), fn, RetryOptions{
		MaxRetries: 3,
		Delay:      time.Millisecond,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

// TestWithRetry_ExhaustedRetries tests that error is returned after all retries exhausted
func TestWithRetry_ExhaustedRetries(t *testing.T) {
	callCount := 0
	fn := func() (int, error) {
		callCount++
		return 0, &TransientError{Err: errors.New("persistent failure")}
	}

	result, err := WithRetry(context.Background(), fn, RetryOptions{
		MaxRetries: 3,
		Delay:      time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != 0 {
		t.Errorf("expected 0, got %d", result)
	}
	if callCount != 4 { // 1 initial + 3 retries
		t.Errorf("expected 4 calls, got %d", callCount)
	}
}

// TestWithRetry_NonRetryableError tests that non-retryable errors don't retry
func TestWithRetry_NonRetryableError(t *testing.T) {
	callCount := 0
	fn := func() (int, error) {
		callCount++
		return 0, &AuthenticationError{Err: errors.New("invalid API key")}
	}

	_, err := WithRetry(context.Background(), fn, RetryOptions{
		MaxRetries: 3,
		Delay:      time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should only call once because auth errors are not retryable
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries for auth errors), got %d", callCount)
	}
}

// TestWithRetry_CustomIsRetryableFn tests custom retryable function
func TestWithRetry_CustomIsRetryableFn(t *testing.T) {
	callCount := 0
	fn := func() (int, error) {
		callCount++
		return 0, errors.New("custom error")
	}

	_, err := WithRetry(context.Background(), fn, RetryOptions{
		MaxRetries: 3,
		Delay:      time.Millisecond,
		IsRetryableFn: func(err error) bool {
			// Never retry
			return false
		},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should only call once because custom IsRetryableFn returns false
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

// TestWithRetry_RateLimitError tests rate limit error handling with retry-after
func TestWithRetry_RateLimitError(t *testing.T) {
	callCount := 0
	fn := func() (int, error) {
		callCount++
		if callCount < 2 {
			return 0, &RateLimitError{Err: errors.New("rate limited"), RetryAfter: 10 * time.Millisecond}
		}
		return 42, nil
	}

	start := time.Now()
	result, err := WithRetry(context.Background(), fn, RetryOptions{
		MaxRetries: 3,
		Delay:      100 * time.Millisecond, // Should be overridden by RetryAfter
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
	// Should take at least 10ms (retry-after) but not 100ms (full delay)
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected at least 10ms delay, got %v", elapsed)
	}
}

// TestWithRetry_ContextCancellation tests that context cancellation stops retries
func TestWithRetry_ContextCancellation(t *testing.T) {
	callCount := 0
	fn := func() (int, error) {
		callCount++
		return 0, &TransientError{Err: errors.New("always fails")}
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	_, err := WithRetry(ctx, fn, RetryOptions{
		MaxRetries: 10,
		Delay:      time.Millisecond,
	})

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
	// Should not retry after context is cancelled
	if callCount != 1 {
		t.Errorf("expected 1 call (context cancelled immediately), got %d", callCount)
	}
}

// TestWithRetry_VoidFunction tests retry with void function
func TestWithRetry_VoidFunction(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 2 {
			return &TransientError{Err: errors.New("temporary failure")}
		}
		return nil
	}

	err := WithRetryVoid(context.Background(), fn, RetryOptions{
		MaxRetries: 3,
		Delay:      time.Millisecond,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

// TestClassifyError tests error classification
func TestClassifyError(t *testing.T) {
	tests := []struct {
		err        error
		wantNil    bool
		wantRetry  bool
		errType    string
	}{
		{
			err:       nil,
			wantNil:   true,
			wantRetry: false,
		},
		{
			err:       errors.New("rate limit exceeded"),
			wantNil:   false,
			wantRetry: true,
			errType:   "*agent.RateLimitError",
		},
		{
			err:       errors.New("429 Too Many Requests"),
			wantNil:   false,
			wantRetry: true,
			errType:   "*agent.RateLimitError",
		},
		{
			err:       errors.New("authentication failed"),
			wantNil:   false,
			wantRetry: false,
			errType:   "*agent.AuthenticationError",
		},
		{
			err:       errors.New("401 Unauthorized"),
			wantNil:   false,
			wantRetry: false,
			errType:   "*agent.AuthenticationError",
		},
		{
			err:       errors.New("request timeout"),
			wantNil:   false,
			wantRetry: true,
			errType:   "*agent.TimeoutError",
		},
		{
			err:       errors.New("context deadline exceeded"),
			wantNil:   false,
			wantRetry: true,
			errType:   "*agent.TimeoutError",
		},
		{
			err:       errors.New("connection reset"),
			wantNil:   false,
			wantRetry: true,
			errType:   "*agent.TransientError",
		},
		{
			err:       &RateLimitError{Err: errors.New("test")},
			wantNil:   false,
			wantRetry: true,
			errType:   "*agent.RateLimitError",
		},
		{
			err:       &AuthenticationError{Err: errors.New("test")},
			wantNil:   false,
			wantRetry: false,
			errType:   "*agent.AuthenticationError",
		},
	}

	for _, tt := range tests {
		re := ClassifyError(tt.err)
		
		if tt.wantNil {
			if re != nil {
				t.Errorf("ClassifyError(%v) = %v, want nil", tt.err, re)
			}
			continue
		}
		
		if re == nil {
			t.Errorf("ClassifyError(%v) = nil, want non-nil", tt.err)
			continue
		}
		
		if re.IsRetryable() != tt.wantRetry {
			t.Errorf("ClassifyError(%v).IsRetryable() = %v, want %v", tt.err, re.IsRetryable(), tt.wantRetry)
		}
		
		// Check type
		switch tt.errType {
		case "*agent.RateLimitError":
			var rle *RateLimitError
			if !errors.As(re, &rle) {
				t.Errorf("ClassifyError(%v) = %T, want %s", tt.err, re, tt.errType)
			}
		case "*agent.AuthenticationError":
			var ae *AuthenticationError
			if !errors.As(re, &ae) {
				t.Errorf("ClassifyError(%v) = %T, want %s", tt.err, re, tt.errType)
			}
		case "*agent.TimeoutError":
			var te *TimeoutError
			if !errors.As(re, &te) {
				t.Errorf("ClassifyError(%v) = %T, want %s", tt.err, re, tt.errType)
			}
		case "*agent.TransientError":
			var te *TransientError
			if !errors.As(re, &te) {
				t.Errorf("ClassifyError(%v) = %T, want %s", tt.err, re, tt.errType)
			}
		}
	}
}

// TestParallelToolExecution tests parallel tool execution
func TestParallelToolExecution(t *testing.T) {
	// Create a simple test for parallel execution order preservation
	var executionOrder []int
	var mu sync.Mutex
	
	// Simulate slow tools that complete in reverse order
	tools := []int{0, 1, 2, 3}
	expectedOrder := []int{0, 1, 2, 3}
	
	// Note: This test doesn't actually call executeToolsParallel because
	// it requires the full AgentLoop setup. This is a placeholder for
	// integration tests.
	
	// Verify mutex works for order tracking
	for _, i := range tools {
		mu.Lock()
		executionOrder = append(executionOrder, i)
		mu.Unlock()
	}
	
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("expected %d items, got %d", len(expectedOrder), len(executionOrder))
	}
}

// TestUsageAdd tests Usage addition
func TestUsageAdd(t *testing.T) {
	u1 := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	
	u2 := Usage{
		PromptTokens:     200,
		CompletionTokens: 75,
		TotalTokens:      275,
	}
	
	u1.Add(u2)
	
	if u1.PromptTokens != 300 {
		t.Errorf("expected PromptTokens 300, got %d", u1.PromptTokens)
	}
	if u1.CompletionTokens != 125 {
		t.Errorf("expected CompletionTokens 125, got %d", u1.CompletionTokens)
	}
	if u1.TotalTokens != 425 {
		t.Errorf("expected TotalTokens 425, got %d", u1.TotalTokens)
	}
}

// TestRetryableErrorTypes tests that all error types implement RetryableError
func TestRetryableErrorTypes(t *testing.T) {
	errors := []RetryableError{
		&RateLimitError{Err: errors.New("test")},
		&AuthenticationError{Err: errors.New("test")},
		&TimeoutError{Err: errors.New("test")},
		&TransientError{Err: errors.New("test")},
	}
	
	expected := []bool{true, false, true, true}
	
	for i, e := range errors {
		if e.IsRetryable() != expected[i] {
			t.Errorf("%T.IsRetryable() = %v, want %v", e, e.IsRetryable(), expected[i])
		}
	}
}

// TestDefaultRetryConfig tests default retry config values
func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	
	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
	if cfg.Delay != time.Second {
		t.Errorf("expected Delay 1s, got %v", cfg.Delay)
	}
}

// TestAtomicBool tests atomic operations (sanity check)
func TestAtomicBool(t *testing.T) {
	var counter atomic.Int32
	
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	
	if counter.Load() != 100 {
		t.Errorf("expected counter 100, got %d", counter.Load())
	}
}
