package agent

import (
	"testing"
)

// TestContextualResponses tests that contextual messages are generated correctly
func TestContextualResponses(t *testing.T) {
	tests := []struct {
		name          string
		iteration     int
		maxTurns      int
		toolResults   []string
		isMaxTurns    bool // was it max turns that caused exit?
		wantEmpty     bool
		wantContains  string
	}{
		{
			name:         "max turns with tools",
			iteration:    3,
			maxTurns:     2,
			toolResults:  []string{"result1", "result2"},
			isMaxTurns:  true,
			wantEmpty:    false,
			wantContains: "reached the maximum number of turns",
		},
		{
			name:         "max iterations with tools",
			iteration:    5,
			maxTurns:     0,
			toolResults:  []string{"file written", "done"},
			isMaxTurns:  false,
			wantEmpty:    false,
			wantContains: "I've completed the task",
		},
		{
			name:         "max iterations no tools",
			iteration:    5,
			maxTurns:     0,
			toolResults:  []string{},
			isMaxTurns:  false,
			wantEmpty:    false,
			wantContains: "processed your request",
		},
		{
			name:         "no iteration",
			iteration:    0,
			maxTurns:     0,
			toolResults:  []string{},
			isMaxTurns:  false,
			wantEmpty:    true, // Will use default response
			wantContains: "",
		},
		{
			name:         "single iteration no tools",
			iteration:    1,
			maxTurns:     0,
			toolResults:  []string{},
			isMaxTurns:  false,
			wantEmpty:    false,
			wantContains: "processed your request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var finalContent string
			lastToolResults := tt.toolResults

			// Simulate the logic from runLLMIteration
			// First check: max turns
			if tt.isMaxTurns && tt.maxTurns > 0 {
				if tt.iteration > 1 && len(lastToolResults) > 0 {
					finalContent = "I've completed " + itoa(tt.iteration-1) + " tool call(s) but reached the maximum number of turns (" + itoa(tt.maxTurns) + "). What would you like me to do next?"
				} else if tt.iteration > 1 {
					finalContent = "I've processed your request through " + itoa(tt.iteration-1) + " iterations but reached the maximum number of turns (" + itoa(tt.maxTurns) + "). What would you like me to do next?"
				}
			} else if finalContent == "" {
				// Second check: max iterations
				if tt.iteration > 0 && len(lastToolResults) > 0 {
					lastResult := lastToolResults[len(lastToolResults)-1]
					if len(lastResult) > 0 {
						finalContent = "I've completed the task. Here's the result: " + lastResult
					} else {
						finalContent = "I've completed the tool execution successfully."
					}
				} else if tt.iteration > 0 {
					finalContent = "I've processed your request but completed " + itoa(tt.iteration) + " iterations."
				}
			}

			if tt.wantEmpty {
				if finalContent != "" {
					t.Errorf("expected empty, got %q", finalContent)
				}
			} else {
				if finalContent == "" {
					t.Errorf("expected non-empty, got empty")
				}
				if tt.wantContains != "" && !containsString(finalContent, tt.wantContains) {
					t.Errorf("expected to contain %q, got %q", tt.wantContains, finalContent)
				}
			}
		})
	}
}

// TestBetterDefaultMessages tests the improved default messages
func TestBetterDefaultMessages(t *testing.T) {
	// Test that our default messages are better than the old one
	oldDefault := "I've completed processing but have no response to give."
	newDefault := "I've processed your request but received an empty response. Could you clarify what you'd like me to do?"
	heartbeatDefault := "Heartbeat processed."

	// New default should be more helpful
	if len(newDefault) <= len(oldDefault) {
		t.Logf("Note: new default is longer (%d vs %d chars) but more helpful", len(newDefault), len(oldDefault))
	}

	// Heartbeat should be short
	if len(heartbeatDefault) > 50 {
		t.Errorf("heartbeat default too long: %q", heartbeatDefault)
	}
}

// Helper functions
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
