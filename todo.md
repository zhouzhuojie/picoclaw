# Agent Loop Improvements Plan

## Overview
Improve `loop.go` based on best practices from pi-mono's agent-loop.ts. This adds production-ready features: max turns, retries, parallel tools, auto mode, token streaming, and usage reporting.

---

## Phase 1: Configuration & Types ✅ DONE

- [x] **1.1** Add `MaxTurns` to `AgentLoopConfig` (default: 0 = unlimited)
- [x] **1.2** Add `MaxRetries` to `AgentLoopConfig` (default: 3)
- [x] **1.3** Add `RetryDelay` to `AgentLoopConfig` (default: 1s)
- [x] **1.4** Add `AutoMode` to `AgentLoopConfig` (default: false) - auto-approve tool calls without user confirmation
- [x] **1.5** Add `ParallelTools` to `AgentLoopConfig` (default: false) - sequential by default to match pi-mono

**Defaults match pi-mono**: ParallelTools=false (sequential), AutoMode=false, MaxTurns=0 (unlimited)

---

## Phase 2: Retry Mechanism ✅ DONE

- [x] **2.1** Create `retry.go` with generic retry helper: `WithRetry(fn, maxRetries, delay, isRetryableError)`
- [x] **2.2** Wrap LLM calls in `runLoop` with retry logic
- [x] **2.3** Wrap tool executions with retry logic
- [x] **2.4** Add error classification: `RateLimitError`, `AuthenticationError`, `TimeoutError`, `TransientError`
- [ ] **2.5** Emit `retry_attempt` event on each retry (event type: `agent_retry`)

---

## Phase 3: Turn Tracking & Max Turns ✅ DONE

- [x] **3.1** Add `turnCount` tracking in `runLoop`
- [ ] **3.2** Emit `turn_start` with turn number: `{ type: "turn_start", turn: number }`
- [x] **3.3** Check `MaxTurns` limit after each turn; stop with `stopReason: "max_turns"` when reached
- [ ] **3.4** Add `maxTurnsReached` to `AgentEvent` type

---

## Phase 4: Parallel Tool Execution ✅ DONE

- [x] **4.1** Modify `executeToolCalls` to run tools concurrently when `ParallelTools` is enabled
- [x] **4.2** Handle tool execution ordering: results must maintain same order as tool calls
- [x] **4.3** If one tool fails in parallel mode, still execute remaining tools (don't fail fast)
- [x] **4.4** Collect all results and push to stream in correct order
- [ ] **4.5** Add `tool_execution_parallel` event: `{ type: "tool_execution_parallel", toolCallIds: string[] }`

---

## Phase 5: Auto Mode ✅ DONE

- [x] **5.1** When `AutoMode: true`, skip user confirmation for tool calls
- [x] **5.2** Execute tool calls immediately without waiting for approval callback
- [ ] **5.3** Still emit `tool_approval_request` event for UI to show (but don't block)
- [ ] **5.4** Add `approved: true` to tool execution events in auto mode

---

## Phase 6: Token-Level Streaming

- [ ] **6.1** Add `token_start`, `token_delta`, `token_end` event types to `AgentEvent`
- [ ] **6.2** Modify `streamAssistantResponse` to emit individual tokens
- [ ] **6.3** For text: emit each text chunk as token
- [ ] **6.4** For reasoning: emit reasoning tokens separately

---

## Phase 7: Usage Reporting ✅ DONE

- [x] **7.1** Extract usage from LLM response (prompt_tokens, completion_tokens)
- [x] **7.2** Log usage at agent end
- [x] **7.3** Add `Usage` type with `PromptTokens`, `CompletionTokens`, `TotalTokens`
- [x] **7.4** Aggregate usage across all turns
- [ ] **7.5** Add `CachedTokens` support (requires provider changes)

---

## Phase 8: Reasoning Events (Explicit)

- [ ] **8.1** Add `reasoning_start`, `reasoning_delta`, `reasoning_end` to `AgentEvent`
- [ ] **8.2** Parse reasoning content from LLM response
- [ ] **8.3** Emit reasoning events separately from text events

---

## Phase 9: Testing ✅ DONE

- [x] **9.1** Add unit tests for retry logic in `retry_test.go`
- [x] **9.2** Add unit tests for parallel tool execution (`loop_features_test.go`)
- [x] **9.3** Add tests for max turns enforcement
- [x] **9.4** Add tests for auto mode behavior
- [x] **9.5** Test usage reporting accuracy

---

## Phase 10: Documentation

- [ ] **10.1** Update `AgentLoopConfig` type comments with new fields
- [ ] **10.2** Add examples to loop.go showing retry, auto mode, parallel tools usage

---

## Implemented Features Summary

| Feature | Status | pi-mono | Default |
|---------|--------|---------|---------|
| MaxTurns | ✅ | ❌ | 0 (unlimited) |
| MaxRetries | ✅ | ❌ | 3 |
| RetryDelay | ✅ | ❌ | 1s |
| AutoMode | ✅ | ❌ | false |
| ParallelTools | ✅ | ❌ (sequential) | false |
| Usage Tracking | ✅ | ❌ | - |

---

## Files Created/Modified

- `pkg/agent/loop.go` - Main implementation
- `pkg/agent/types.go` - Usage and LoopConfig types
- `pkg/agent/retry.go` - Retry helper with error classification
- `pkg/agent/retry_test.go` - Retry unit tests
- `pkg/agent/loop_features_test.go` - Feature unit tests
- `pkg/config/config.go` - New config fields
