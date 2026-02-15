# Agent Loop Improvements

## Completed Features ✅

| Feature | Default | Notes |
|---------|---------|-------|
| MaxTurns | 0 (unlimited) | New - pi-mono doesn't have |
| MaxRetries | 3 | New - pi-mono doesn't have |
| RetryDelay | 1s | New - pi-mono doesn't have |
| AutoMode | false | New - pi-mono doesn't have |
| ParallelTools | false | Matches pi-mono (sequential) |
| Usage Tracking | - | New - pi-mono doesn't have |

---

## Remaining Tasks

### P0 - Critical: Fix Empty Response Issue ✅ DONE

- [x] **P0.1** When max turns reached: return contextual message instead of empty
- [x] **P0.2** When max iterations reached: return message about executed tools
- [x] **P0.3** When LLM returns empty: try regeneration or use tool results
- [x] **P0.4** Add contextual default messages based on what happened

### P1 - Nice to Have

- [ ] **P1.1** Emit `retry_attempt` event on each retry
- [ ] **P1.2** Emit `turn_start` with turn number
- [ ] **P1.3** Emit `tool_execution_parallel` event
- [ ] **P1.4** Token-level streaming
- [ ] **P1.5** Reasoning events

---

## Files

- `pkg/agent/loop.go` - Main implementation
- `pkg/agent/retry.go` - Retry helper
- `pkg/agent/types.go` - Usage, LoopConfig types
- `pkg/config/config.go` - Config fields
