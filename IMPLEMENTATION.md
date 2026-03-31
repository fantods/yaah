# Implementation Plan

Test-first, many small phases. Each phase: write tests, then implement, then verify tests pass. Every phase produces a compilable, testable increment.


## Phase 5 — Provider Options + Context (`internal/provider/options.go`, `context.go`)

| | |
|---|---|
| **Test** | `ThinkingLevel` enum, `ThinkingBudgets`, `StreamOptions` construction, `Context` construction |
| **Implement** | `ThinkingLevel`, `ThinkingBudgets`, `StreamOptions`, `SimpleStreamOptions`, `Context` |

---

## Phase 6 — Provider Event Types (`internal/provider/events.go`)

| | |
|---|---|
| **Test** | Each event type satisfies `AssistantMessageEvent` interface, field access |
| **Implement** | All 11 event types + interface |

---

## Phase 7 — EventStream (`internal/provider/eventstream.go`)

| | |
|---|---|
| **Test** | Push/Events/End flow, `Result` blocking, concurrent `Push`, stream completion, overflow protection |
| **Implement** | `EventStream[T, R]` generic struct |

> This is the most critical piece — needs thorough concurrent testing.

---

## Phase 8 — Provider Registry (`internal/provider/registry.go`)

| | |
|---|---|
| **Test** | `Register`, `Lookup` found/not-found, concurrent access, overwrite |
| **Implement** | `sync.Map`-based registry with `Register`/`Lookup` |

---

## Phase 9 — Partial JSON Parser (`internal/provider/partial_json.go`)

| | |
|---|---|
| **Test** | Incremental parsing, complete/incomplete JSON, edge cases |
| **Implement** | Streaming JSON parser for tool call arguments |

---

## Phase 10 — Cross-Provider Transforms (`internal/provider/transform.go`)

| | |
|---|---|
| **Test** | Tool call ID normalization, thinking block handling, message transformation |
| **Implement** | Transform functions |

---

## Phase 11 — Simple Options Helpers (`internal/provider/simple_options.go`)

| | |
|---|---|
| **Test** | `BuildBaseOptions`, `AdjustMaxTokensForThinking` |
| **Implement** | Shared option helpers |

---

## Phase 12 — Anthropic Provider: Convert (`internal/provider/anthropic/convert.go`)

| | |
|---|---|
| **Test** | `Message` → Anthropic SDK type conversion (table-driven), tool conversion |
| **Implement** | `convertMessages`, `convertTools` |

---

## Phase 13 — Anthropic Provider: Parse (`internal/provider/anthropic/parse.go`)

| | |
|---|---|
| **Test** | SSE event parsing, delta handling |
| **Implement** | `parseEvent`, `handleDelta` functions |

---

## Phase 14 — Anthropic Provider: Stream (`internal/provider/anthropic/anthropic.go`)

| | |
|---|---|
| **Test** | Stream function returns correct `EventStream`, event sequence matches expected flow |
| **Implement** | `StreamAnthropic`, `StreamSimpleAnthropic` |

---

## Phase 15 — Anthropic Provider: Options (`internal/provider/anthropic/options.go`)

| | |
|---|---|
| **Test** | Cache control, stealth mode option construction |
| **Implement** | `AnthropicOptions`, cache control, stealth mode |

---

## Phase 16 — OpenAI Provider: Convert (`internal/provider/openai/convert.go`)

| | |
|---|---|
| **Test** | `Message` → OpenAI SDK type conversion, tool conversion |
| **Implement** | `convertMessages`, `convertTools` |

---

## Phase 17 — OpenAI Provider: Parse (`internal/provider/openai/parse.go`)

| | |
|---|---|
| **Test** | Chunk parsing, tool call delta assembly, reasoning |
| **Implement** | `parseChunk`, `assembleToolCallDeltas` |

---

## Phase 18 — OpenAI Provider: Stream (`internal/provider/openai/openai.go`)

| | |
|---|---|
| **Test** | Stream function, event sequence |
| **Implement** | `StreamOpenAICompletions`, `StreamSimpleOpenAICompletions` |

---

## Phase 19 — OpenAI Provider: Options + Responses (`internal/provider/openai/options.go`, `responses.go`)

| | |
|---|---|
| **Test** | Compat flags, reasoning effort mapping, responses API variant |
| **Implement** | OpenAI options, Responses API |

---

## Phase 20 — z.ai Provider (`internal/provider/zai/`)

| | |
|---|---|
| **Test** | z.ai-specific conversion, options |
| **Implement** | `StreamZai`, `StreamSimpleZai`, convert, options |

---

## Phase 21 — Register Builtins (`internal/provider/register.go`)

| | |
|---|---|
| **Test** | `RegisterBuiltins` registers all providers, `Lookup` succeeds for each |
| **Implement** | `RegisterBuiltins` |

---

## Phase 22 — Agent Event Types (`internal/agent/events.go`)

| | |
|---|---|
| **Test** | Each event satisfies `AgentEvent` interface |
| **Implement** | All 9 event types + interface |

---

## Phase 23 — Agent Tool Interface (`internal/agent/tool.go`)

| | |
|---|---|
| **Test** | `MockTool` implements interface, `AgentToolResult` construction |
| **Implement** | `AgentTool`, `AgentToolResult`, `AgentToolCall` |

---

## Phase 24 — Agent Options (`internal/agent/options.go`)

| | |
|---|---|
| **Test** | Options construction, hook types |
| **Implement** | `AgentOptions`, `AgentLoopConfig`, `BeforeToolCallContext`, `AfterToolCallContext`, hook types |

---

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
