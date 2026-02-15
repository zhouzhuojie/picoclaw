package agent

// Usage represents token usage from an LLM response
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Add adds another Usage to this one
func (u *Usage) Add(other Usage) {
	u.PromptTokens += other.PromptTokens
	u.CompletionTokens += other.CompletionTokens
	u.TotalTokens += other.TotalTokens
}

// StopReason indicates why the agent loop ended
type StopReason string

const (
	StopReasonEnd         StopReason = "end"
	StopReasonError       StopReason = "error"
	StopReasonAborted     StopReason = "aborted"
	StopReasonMaxTurns    StopReason = "max_turns"
	StopReasonMaxIter     StopReason = "max_iterations"
)

// LoopConfig holds runtime loop configuration (derived from AgentDefaults)
type LoopConfig struct {
	MaxTurns      int     // Max turns (0 = unlimited)
	MaxRetries    int     // Max retries
	RetryDelay    int     // Retry delay in seconds
	ParallelTools bool    // Parallel tool execution
	// LLM defaults - can be overridden per request
	MaxTokens   int     // Max tokens for LLM response (default: 8192)
	Temperature float64 // Temperature for LLM (default: 0.7)
}

// DefaultLoopConfig returns default loop configuration
func DefaultLoopConfig() LoopConfig {
	return LoopConfig{
		MaxTurns:      0,
		MaxRetries:    3,
		RetryDelay:    1,
		ParallelTools: false,
		MaxTokens:     8192,
		Temperature:   0.7,
	}
}
