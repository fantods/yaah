# Implementation Plan

Test-first, many small phases. Each phase: write tests, then implement, then verify tests pass. Every phase produces a compilable, testable increment.

## Phase 25 — Agent State (`internal/agent/state.go`)

| | |
|---|---|
| **Test** | New state, state mutations (set streaming, add message, track tool calls, error propagation) |
| **Implement** | `AgentState`, mutation methods |

---

## Phase 26 — Pending Message Queue (`internal/agent/queue.go`)

| | |
|---|---|
| **Test** | Enqueue/dequeue, steering vs follow-up, queue modes, concurrent access |
| **Implement** | `PendingMessageQueue`, `QueueMode` |

---

## Phase 27 — Agent Loop: Core Skeleton (`internal/agent/loop.go`)

| | |
|---|---|
| **Test** | `AgentLoop` emits correct event sequence for simple text-only turn (no tool calls), mock `StreamFn` |
| **Implement** | `AgentLoop`, `streamAssistantResponse` with mock support |

---

## Phase 28 — Agent Loop: Tool Calls (`internal/agent/loop.go`)

| | |
|---|---|
| **Test** | Loop with tool calls emits correct events, handles multiple rounds |
| **Implement** | `runLoop` with tool call handling |

---

## Phase 29 — Tool Execution: Sequential (`internal/agent/tool_exec.go`)

| | |
|---|---|
| **Test** | Sequential execution with mock tools, before/after hooks, error handling |
| **Implement** | `executeToolCalls` sequential mode |

---

## Phase 30 — Tool Execution: Parallel (`internal/agent/tool_exec.go`)

| | |
|---|---|
| **Test** | Parallel execution, result ordering, concurrent safety |
| **Implement** | `executeToolCalls` parallel mode |

---

## Phase 31 — Agent Struct (`internal/agent/agent.go`)

| | |
|---|---|
| **Test** | `NewAgent`, `Prompt`, `Subscribe`, `Abort`, `Steer`, `FollowUp`, `State` |
| **Implement** | `Agent` struct with all public methods |

---

## Phase 32 — Agent Proxy (`internal/agent/proxy.go`)

| | |
|---|---|
| **Test** | Proxy stream function, request routing |
| **Implement** | `StreamProxy` |

---

## Phase 33 — Agent Loop: Steering + Follow-Up (`internal/agent/loop.go`)

| | |
|---|---|
| **Test** | Steering interrupts mid-loop, follow-up continues after completion |
| **Implement** | Steering/follow-up message injection in `runLoop` |

---

## Phase 34 — TUI: Theme (`internal/tui/theme.go`)

| | |
|---|---|
| **Test** | Theme color values, style construction |
| **Implement** | `Theme` struct, color constants, style helpers |

---

## Phase 35 — TUI: Key Bindings (`internal/tui/keymap.go`)

| | |
|---|---|
| **Test** | Default key bindings present, custom bindings override |
| **Implement** | `KeyMap` struct, `DefaultKeyMap` |

---

## Phase 36 — TUI: Event Bridge (`internal/tui/messages.go`)

| | |
|---|---|
| **Test** | `agentEventMsg` wrapping, `waitForAgentEvents` cmd |
| **Implement** | `tea.Msg` types, bridge function |

---

## Phase 37 — TUI: Input Component (`internal/tui/input.go`)

| | |
|---|---|
| **Test** | Init/Update/View, Enter sends, Shift+Enter newline |
| **Implement** | `inputModel` with textarea |

---

## Phase 38 — TUI: Chat + Streaming (`internal/tui/chat.go`, `streaming.go`)

| | |
|---|---|
| **Test** | Viewport scroll, auto-scroll, delta accumulation |
| **Implement** | `chatModel`, `streamingModel` |

---

## Phase 39 — TUI: Thinking + Tools + Status (`internal/tui/thinking.go`, `tools.go`, `status.go`)

| | |
|---|---|
| **Test** | Thinking block toggle, tool execution display, status bar values |
| **Implement** | `thinkingModel`, `toolsModel`, `statusModel` |

---

## Phase 40 — TUI: App Root (`internal/tui/app.go`)

| | |
|---|---|
| **Test** | Component composition, event routing, full render cycle |
| **Implement** | `appModel` wiring all components |

---

## Phase 41 — Entrypoint (`main.go`)

| | |
|---|---|
| **Test** | Build succeeds, minimal smoke test |
| **Implement** | `main.go` wiring agent + TUI |
