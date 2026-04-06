# Implementation Plan

Test-first, many small phases. Each phase: write tests, then implement, then verify tests pass. Every phase produces a compilable, testable increment.

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
