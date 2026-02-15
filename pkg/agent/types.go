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
	MaxTurns      int           // Max turns (0 = unlimited)
	MaxRetries   int           // Max retries
	RetryDelay   int           // Retry delay in seconds
	AutoMode     bool          // Auto-approve tool calls
	ParallelTools bool         // Parallel tool execution
}
