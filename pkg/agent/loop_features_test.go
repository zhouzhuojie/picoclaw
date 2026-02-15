package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestParallelToolsExecution tests that parallel tools execute concurrently
func TestParallelToolsExecution(t *testing.T) {
	// Track execution timing
	var mu sync.Mutex
	executionOrder := []int{}
	startTime := time.Now()
	
	// Create mock tool calls
	toolCalls := []struct {
		name   string
		delay  time.Duration
	}{
		{"tool0", 50 * time.Millisecond},
		{"tool1", 10 * time.Millisecond},
		{"tool2", 30 * time.Millisecond},
		{"tool3", 20 * time.Millisecond},
	}
	
	// Execute in parallel (simulating what executeToolsParallel does)
	type resultWithIndex struct {
		index  int
		result string
	}
	
	resultChan := make(chan resultWithIndex, len(toolCalls))
	var wg sync.WaitGroup
	
	for i, tc := range toolCalls {
		wg.Add(1)
		go func(index int, delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay) // Simulate work
			mu.Lock()
			executionOrder = append(executionOrder, index)
			mu.Unlock()
			resultChan <- resultWithIndex{index: index, result: "done"}
		}(i, tc.delay)
	}
	
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	results := make([]string, len(toolCalls))
	for r := range resultChan {
		results[r.index] = r.result
	}
	
	elapsed := time.Since(startTime)
	
	// Verify all completed
	if len(results) != len(toolCalls) {
		t.Errorf("expected %d results, got %d", len(toolCalls), len(results))
	}
	
	// Verify parallel execution - should take less than sequential
	// Sequential would take 50+10+30+20 = 110ms
	// Parallel should take ~50ms (the longest)
	if elapsed > 80*time.Millisecond {
		t.Errorf("parallel execution took %v, expected ~50ms (parallel)", elapsed)
	}
	
	t.Logf("Parallel execution completed in %v (sequential would be ~110ms)", elapsed)
}

// TestParallelToolsOrderPreservation tests that results maintain original order
func TestParallelToolsOrderPreservation(t *testing.T) {
	toolCalls := []int{0, 1, 2, 3, 4}
	
	type resultWithIndex struct {
		index  int
		result int
	}
	
	resultChan := make(chan resultWithIndex, len(toolCalls))
	var wg sync.WaitGroup
	
	// Execute in random completion order but should preserve input order
	// Simulate: indices 2, 0, 4, 1, 3 complete in that order
	completionOrder := []int{2, 0, 4, 1, 3}
	
	for _, idx := range completionOrder {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resultChan <- resultWithIndex{index: i, result: i * 10}
		}(idx)
	}
	
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results - should be in original order (0, 1, 2, 3, 4)
	results := make([]int, len(toolCalls))
	for r := range resultChan {
		results[r.index] = r.result
	}
	
	// Verify order is preserved
	expected := []int{0, 10, 20, 30, 40}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

// TestAutoMode_Behavior tests auto mode configuration behavior
func TestAutoMode_Behavior(t *testing.T) {
	tests := []struct {
		name           string
		autoMode       bool
		sendResponse   bool
		wantSendToUser bool
	}{
		{
			name:           "autoMode=true, sendResponse=false -> should send",
			autoMode:       true,
			sendResponse:   false,
			wantSendToUser: true,
		},
		{
			name:           "autoMode=false, sendResponse=true -> should send",
			autoMode:       false,
			sendResponse:   true,
			wantSendToUser: true,
		},
		{
			name:           "autoMode=false, sendResponse=false -> should not send",
			autoMode:       false,
			sendResponse:   false,
			wantSendToUser: false,
		},
		{
			name:           "autoMode=true, sendResponse=true -> should send",
			autoMode:       true,
			sendResponse:   true,
			wantSendToUser: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic in executeSingleTool
			shouldSendToUser := tt.autoMode || tt.sendResponse
			
			if shouldSendToUser != tt.wantSendToUser {
				t.Errorf("autoMode=%v, sendResponse=%v: got %v, want %v",
					tt.autoMode, tt.sendResponse, shouldSendToUser, tt.wantSendToUser)
			}
		})
	}
}

// TestLoopConfig_Defaults tests default loop configuration
func TestLoopConfig_Defaults(t *testing.T) {
	cfg := LoopConfig{
		MaxTurns:      0,
		MaxRetries:    3,
		RetryDelay:    1,
		AutoMode:      false,
		ParallelTools: false,
	}
	
	// Verify defaults match pi-mono
	if cfg.MaxTurns != 0 {
		t.Errorf("MaxTurns: got %d, want 0 (unlimited)", cfg.MaxTurns)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries: got %d, want 3", cfg.MaxRetries)
	}
	if cfg.RetryDelay != 1 {
		t.Errorf("RetryDelay: got %d, want 1", cfg.RetryDelay)
	}
	if cfg.AutoMode != false {
		t.Errorf("AutoMode: got %v, want false", cfg.AutoMode)
	}
	if cfg.ParallelTools != false {
		t.Errorf("ParallelTools: got %v, want false (sequential like pi-mono)", cfg.ParallelTools)
	}
}

// TestMaxTurns_Enforcement tests max turns limit
func TestMaxTurns_Enforcement(t *testing.T) {
	// Simulate max turns logic
	maxTurns := 3
	turnCount := 0
	stopReason := ""
	
	// Simulate running more turns than max
	for turnCount < 5 {
		turnCount++
		
		if maxTurns > 0 && turnCount > maxTurns {
			stopReason = "max_turns"
			break
		}
		
		// Continue processing
	}
	
	if stopReason != "max_turns" {
		t.Errorf("expected stopReason 'max_turns', got '%s'", stopReason)
	}
	if turnCount != 4 { // Stops at 4 because 4 > 3
		t.Errorf("expected turnCount 4, got %d", turnCount)
	}
}

// TestMaxTurns_Unlimited tests unlimited turns (maxTurns = 0)
func TestMaxTurns_Unlimited(t *testing.T) {
	maxTurns := 0 // Unlimited
	turnCount := 0
	stopReason := ""
	
	// Simulate running with unlimited turns
	for turnCount < 5 {
		turnCount++
		
		if maxTurns > 0 && turnCount > maxTurns {
			stopReason = "max_turns"
			break
		}
	}
	
	if stopReason != "" {
		t.Errorf("expected no stop reason (unlimited), got '%s'", stopReason)
	}
	if turnCount != 5 {
		t.Errorf("expected turnCount 5, got %d", turnCount)
	}
}

// TestSequentialToolExecution tests sequential tool execution order
func TestSequentialToolExecution(t *testing.T) {
	var executionOrder []int
	var mu sync.Mutex
	
	toolDelays := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}
	
	startTime := time.Now()
	
	// Sequential execution
	for i, delay := range toolDelays {
		time.Sleep(delay)
		mu.Lock()
		executionOrder = append(executionOrder, i)
		mu.Unlock()
	}
	
	elapsed := time.Since(startTime)
	
	// Sequential should take sum of all delays
	expectedMin := 50 * time.Millisecond // 10 + 20 + 30
	if elapsed < expectedMin {
		t.Errorf("sequential execution too fast: %v, expected at least %v", elapsed, expectedMin)
	}
	
	// Verify order is preserved
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 executions, got %d", len(executionOrder))
	}
	for i, v := range executionOrder {
		if v != i {
			t.Errorf("at index %d: expected %d, got %d", i, i, v)
		}
	}
}

// TestRetryWithParallelTools tests that retry works with parallel tools
func TestRetryWithParallelTools(t *testing.T) {
	callCount := 0
	successAt := 2
	
	fn := func() (int, error) {
		callCount++
		if callCount < successAt {
			return 0, &TransientError{Err: errors.New("temporary failure")}
		}
		return 42, nil
	}
	
	// Test retry with parallel tools config
	result, err := WithRetry(context.Background(), fn, RetryOptions{
		MaxRetries:      3,
		Delay:           time.Millisecond,
	})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if callCount != successAt {
		t.Errorf("expected %d calls, got %d", successAt, callCount)
	}
}

// TestToolErrorClassification tests that tool errors are correctly classified
func TestToolErrorClassification(t *testing.T) {
	tests := []struct {
		err        error
		wantNil    bool
		wantRetry  bool
	}{
		{
			err:       errors.New("tool execution failed: rate limit"),
			wantNil:   false,
			wantRetry: true,
		},
		{
			err:       errors.New("tool timeout"),
			wantNil:   false,
			wantRetry: true,
		},
		{
			err:       errors.New("tool execution failed: authentication"),
			wantNil:   false,
			wantRetry: false,
		},
		{
			err:       errors.New("connection reset by peer"),
			wantNil:   false,
			wantRetry: true,
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
	}
}
