# Agent Loop Improvements Plan

## Overview
Improve `loop.go` based on best practices from pi-mono's agent-loop.ts. This adds production-ready features: max turns, retries, parallel tools, auto mode, token streaming, and usage reporting.

---

## Phase 1: Configuration & Types

- [ ] **1.1** Add `MaxTurns` to `AgentLoopConfig` (default: unlimited or 100)
- [ ] **1.2** Add `MaxRetries` to `AgentLoopConfig` (default: 3)
- [ ] **1.3** Add `RetryDelay` to `AgentLoopConfig` (default: 1s)
- [ ] **1.4** Add `AutoMode` to `AgentLoopConfig` (default: false) - auto-approve tool calls without user confirmation
- [ ] **1.5** Add `ParallelTools` to `AgentLoopConfig` (default: true) - enable parallel tool execution

---

## Phase 2: Retry Mechanism

- [ ] **2.1** Create `retry.go` with generic retry helper: `WithRetry(fn, maxRetries, delay, isRetryableError)`
- [ ] **2.2** Wrap LLM calls in `runLoop` with retry logic
- [ ] **2.3** Wrap tool executions with retry logic
- [ ] **2.4** Add error classification: `RateLimitError`, `AuthenticationError`, `TimeoutError`, `TransientError`
- [ ] **2.5** Emit `retry_attempt` event on each retry (event type: `agent_retry`)

---

## Phase 3: Turn Tracking & Max Turns

- [ ] **3.1** Add `turnCount` tracking in `runLoop`
- [ ] **3.2** Emit `turn_start` with turn number: `{ type: "turn_start", turn: number }`
- [ ] **3.3** Check `MaxTurns` limit after each turn; emit `agent_end` with `stopReason: "max_turns"` when reached
- [ ] **3.4** Add `maxTurnsReached` to `AgentEvent` type

---

## Phase 4: Parallel Tool Execution

- [ ] **4.1** Modify `executeToolCalls` to run tools concurrently when `ParallelTools` is enabled
- [ ] **4.2** Handle tool execution ordering: results must maintain same order as tool calls
- [ ] **4.3** If one tool fails in parallel mode, still execute remaining tools (don't fail fast)
- [ ] **4.4** Collect all results and push to stream in correct order
- [ ] **4.5** Add `tool_execution_parallel` event: `{ type: "tool_execution_parallel", toolCallIds: string[] }`

---

## Phase 5: Auto Mode

- [ ] **5.1** When `AutoMode: true`, skip user confirmation for tool calls
- [ ] **5.2** Execute tool calls immediately without waiting for approval callback
- [ ] **5.3** Still emit `tool_approval_request` event for UI to show (but don't block)
- [ ] **5.4** Add `approved: true` to tool execution events in auto mode

---

## Phase 6: Token-Level Streaming

- [ ] **6.1** Add `token_start`, `token_delta`, `token_end` event types to `AgentEvent`
- [ ] **6.2** Modify `streamAssistantResponse` to emit individual tokens
- [ ] **6.3** For text: emit each text chunk as token
- [ ] **6.4** For reasoning: emit reasoning tokens separately

---

## Phase 7: Usage Reporting

- [ ] **7.1** Extract usage from LLM response (prompt_tokens, completion_tokens, cached_tokens)
- [ ] **7.2** Emit `agent_usage` event at `agent_end`: `{ type: "agent_usage", usage: Usage }`
- [ ] **7.3** Add `Usage` type with `PromptTokens`, `CompletionTokens`, `CachedTokens`, `TotalTokens`
- [ ] **7.4** Aggregate usage across all turns

---

## Phase 8: Reasoning Events (Explicit)

- [ ] **8.1** Add `reasoning_start`, `reasoning_delta`, `reasoning_end` to `AgentEvent`
- [ ] **8.2** Parse reasoning content from LLM response
- [ ] **8.3** Emit reasoning events separately from text events

---

## Phase 9: Testing

- [ ] **9.1** Add unit tests for retry logic in `retry_test.go`
- [ ] **9.2** Add unit tests for parallel tool execution
- [ ] **9.3** Add integration tests for max turns enforcement
- [ ] **9.4** Add tests for auto mode behavior
- [ ] **9.5** Test usage reporting accuracy

---

## Phase 10: Documentation

- [ ] **10.1** Update `AgentLoopConfig` type comments with new fields
- [ ] **10.2** Add examples to loop.go showing retry, auto mode, parallel tools usage

---

## Event Types Summary (New + Modified)

| Event | Description |
|-------|-------------|
| `agent_retry` | Retry attempt: `{ type: "agent_retry", attempt: number, error: string }` |
| `turn_start` | Turn started (add `turn: number`) |
| `agent_end` | Add `stopReason: "max_turns"` option |
| `tool_execution_parallel` | Parallel execution: `{ type: "tool_execution_parallel", toolCallIds: string[] }` |
| `token_start` | Token stream start |
| `token_delta` | Token content: `{ type: "token_delta", token: string, tokenType: "text" | "reasoning" }` |
| `token_end` | Token stream end |
| `reasoning_start` | Reasoning start |
| `reasoning_delta` | Reasoning content |
| `reasoning_end` | Reasoning end |
| `agent_usage` | Usage report: `{ type: "agent_usage", usage: Usage }` |

---

## Files to Modify

- `loop.go` - Main implementation
- `types.go` - Add new event types and config fields
- `retry.go` - New file for retry logic
- `loop_test.go` - Add tests

---

## Files to Create

- `retry.go` - Retry helper
- `retry_test.go` - Retry tests
