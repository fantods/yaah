# yaah Implementation Plan

## 1. Project Overview

**Goal:** Build a stateful coding agent in Go with tool execution, event streaming via typed channels, multi-provider support (Anthropic, OpenAI, z.ai), and a bubbletea TUI — architecturally modeled after the `pi-mono/packages/agent` reference.

| | |
|---|---|
| **Go version** | 1.25 |
| **Dependencies** | Official SDKs where available (`github.com/anthropics/anthropic-sdk-go`, `github.com/openai/openai-go`), raw HTTP for providers without SDKs |
| **Event model** | Typed channels with concrete event structs implementing an `AgentEvent` interface |
| **Scope** | Core agent library + provider layer + full bubbletea TUI |

---

## 2. Package Structure

```
yaah/
├── go.mod
├── go.sum
├── main.go                          # Entrypoint, wires TUI + agent
├── PLAN.md
├── AGENTS.md
│
├── internal/
│   ├── message/                     # Core message types (shared across packages)
│   │   ├── message.go               # UserMessage, AssistantMessage, ToolResultMessage, Message union
│   │   ├── content.go               # TextContent, ThinkingContent, ImageContent, ToolCall
│   │   └── usage.go                 # Usage, StopReason, Cost types
│   │
│   ├── provider/                    # Provider abstraction layer
│   │   ├── provider.go              # Provider interface, StreamFn type, registry
│   │   ├── registry.go              # Provider registration, lookup by API name
│   │   ├── options.go               # StreamOptions, SimpleStreamOptions, ThinkingLevel, ThinkingBudgets
│   │   ├── context.go               # Context type (systemPrompt + messages + tools)
│   │   ├── model.go                 # Model struct, KnownProvider constants
│   │   ├── tool.go                  # Tool struct (name, description, JSON schema)
│   │   ├── eventstream.go           # EventStream[T], AssistantMessageEventStream (channel-based)
│   │   ├── events.go                # AssistantMessageEvent union types
│   │   ├── transform.go             # Cross-provider message transforms (tool call ID normalization, thinking block handling)
│   │   ├── partial_json.go          # Streaming JSON parser for tool call arguments
│   │   ├── simple_options.go        # BuildBaseOptions, AdjustMaxTokensForThinking
│   │   │
│   │   ├── anthropic/               # Anthropic Messages API
│   │   │   ├── anthropic.go         # StreamAnthropic, StreamSimpleAnthropic
│   │   │   ├── convert.go           # Message conversion (Message → Anthropic SDK types)
│   │   │   ├── parse.go             # SSE event parsing, delta handling
│   │   │   └── options.go           # Anthropic-specific options (cache, stealth mode)
│   │   │
│   │   ├── openai/                  # OpenAI Chat Completions API
│   │   │   ├── openai.go            # StreamOpenAICompletions, StreamSimpleOpenAICompletions
│   │   │   ├── convert.go           # Message → OpenAI SDK types
│   │   │   ├── parse.go             # Chunk parsing, tool call delta assembly
│   │   │   ├── responses.go         # OpenAI Responses API variant
│   │   │   └── options.go           # OpenAI-specific options (compat flags, reasoning effort)
│   │   │
│   │   ├── zai/                     # z.ai provider (OpenAI-compatible with extensions)
│   │   │   ├── zai.go               # StreamZai, StreamSimpleZai
│   │   │   ├── convert.go           # z.ai-specific message conversion
│   │   │   └── options.go           # z.ai options (tool_stream, enable_thinking)
│   │   │
│   │   └── register.go              # RegisterBuiltinProviders(), called from init or main
│   │
│   ├── agent/                       # Core agent loop + stateful Agent
│   │   ├── agent.go                 # Agent struct, NewAgent(), Prompt(), Continue(), Subscribe(), State
│   │   ├── loop.go                  # AgentLoop(), AgentLoopContinue(), runLoop(), streamAssistantResponse()
│   │   ├── tool_exec.go             # executeToolCalls (sequential + parallel), prepareToolCall, finalizeToolCall
│   │   ├── events.go                # AgentEvent interface + concrete event types (11 event types)
│   │   ├── state.go                 # AgentState, mutable state management
│   │   ├── queue.go                 # PendingMessageQueue (steering + follow-up), QueueMode
│   │   ├── options.go               # AgentOptions, AgentLoopConfig, BeforeToolCall/AfterToolCall hooks
│   │   ├── tool.go                  # AgentTool interface, AgentToolResult, AgentToolCall
│   │   └── proxy.go                 # StreamProxy for backend-routed LLM calls
│   │
│   └── tui/                         # Bubbletea TUI
│       ├── app.go                   # Root bubbletea model, main loop
│       ├── messages.go              # tea.Msg types wrapping agent events
│       ├── input.go                 # Input area component (text area + submit)
│       ├── chat.go                  # Chat/message list component
│       ├── streaming.go             # Streaming text rendering (assistant message accumulation)
│       ├── tools.go                 # Tool execution display (start/end/update)
│       ├── thinking.go              # Thinking block display
│       ├── status.go                # Status bar (model, tokens, cost, state)
│       ├── keymap.go                # Key bindings (configurable)
│       └── theme.go                 # Colors/styles
│
└── cmd/
    └── yaah/
        └── main.go                  # Alternative CLI entrypoint (if not using root main.go)
```

---

## 3. Core Types (Detailed)

### 3.1 Message Types (`internal/message/`)

```go
// content.go
type TextContent struct {
    Type          string  `json:"type"`          // "text"
    Text          string  `json:"text"`
    TextSignature string  `json:"textSignature,omitempty"`
}

type ThinkingContent struct {
    Type               string `json:"type"`               // "thinking"
    Thinking           string `json:"thinking"`
    ThinkingSignature  string `json:"thinkingSignature,omitempty"`
    Redacted           bool   `json:"redacted,omitempty"`
}

type ImageContent struct {
    Type     string `json:"type"`     // "image"
    Data     string `json:"data"`     // base64
    MIMEType string `json:"mimeType"`
}

type ToolCall struct {
    Type             string         `json:"type"`           // "toolCall"
    ID               string         `json:"id"`
    Name             string         `json:"name"`
    Arguments        map[string]any `json:"arguments"`
    ThoughtSignature string         `json:"thoughtSignature,omitempty"`
}
```

```go
// message.go
type UserMessage struct {
    Role      string         `json:"role"`      // "user"
    Content   []ContentBlock `json:"content"`   // TextContent | ImageContent
    Timestamp int64          `json:"timestamp"`
}

type AssistantMessage struct {
    Role         string         `json:"role"`                   // "assistant"
    Content      []ContentBlock `json:"content"`                // TextContent | ThinkingContent | ToolCall
    API          string         `json:"api"`
    Provider     string         `json:"provider"`
    Model        string         `json:"model"`
    ResponseID   string         `json:"responseId,omitempty"`
    Usage        Usage          `json:"usage"`
    StopReason   StopReason     `json:"stopReason"`
    ErrorMessage string         `json:"errorMessage,omitempty"`
    Timestamp    int64          `json:"timestamp"`
}

type ToolResultMessage struct {
    Role       string         `json:"role"`       // "toolResult"
    ToolCallID string         `json:"toolCallId"`
    ToolName   string         `json:"toolName"`
    Content    []ContentBlock `json:"content"`
    Details    any            `json:"details,omitempty"`
    IsError    bool           `json:"isError"`
    Timestamp  int64          `json:"timestamp"`
}

// Message is the union type — any of the three
type Message interface{ messageUnion() }
```

### 3.2 Provider Types (`internal/provider/`)

```go
// model.go
type Model struct {
    ID            string
    Name          string
    API           string          // e.g., "anthropic-messages", "openai-completions"
    Provider      string          // e.g., "anthropic", "openai", "zai"
    BaseURL       string
    Reasoning     bool
    Input         []string        // {"text"}, {"text","image"}
    Cost          ModelCost
    ContextWindow int
    MaxTokens     int
    Headers       map[string]string
    Compat        any             // Provider-specific compat flags
}

type ModelCost struct {
    Input      float64
    Output     float64
    CacheRead  float64
    CacheWrite float64
}
```

```go
// provider.go
type StreamFn func(model Model, ctx Context, opts *StreamOptions) *AssistantMessageEventStream

type Provider struct {
    API          string
    Stream       StreamFn
    StreamSimple StreamFn
}
```

```go
// context.go
type Context struct {
    SystemPrompt string
    Messages     []Message
    Tools        []Tool
}
```

### 3.3 Event Stream (`internal/provider/eventstream.go`)

Go equivalent of the TS `EventStream` class, using channels:

```go
type EventStream[T any, R any] struct {
    ch     chan T
    done   chan struct{}
    result R
    once   sync.Once
}

func NewEventStream[T, R](isComplete func(T) bool, extractResult func(T) R) *EventStream[T, R]
func (s *EventStream[T, R]) Push(event T)
func (s *EventStream[T, R]) End(result *R)
func (s *EventStream[T, R]) Events() <-chan T      // Read-only channel for consumers
func (s *EventStream[T, R]) Result() <-chan R       // Blocks until stream completes

type AssistantMessageEventStream = EventStream[AssistantMessageEvent, AssistantMessage]
```

**Key design decision:** Go doesn't have `AsyncIterable` / `for await`. Instead:

- `Events()` returns a `<-chan T` that consumers `range` over
- `Result()` returns a `<-chan R` (single-value) that resolves when the stream completes
- `Push()` sends to the channel non-blocking (with overflow protection)

### 3.4 Assistant Message Events (`internal/provider/events.go`)

```go
type AssistantMessageEvent interface{ assistantEvent() }

type EventStart         struct { /* partial *AssistantMessage */ }
type EventTextStart     struct { ContentIndex int /* partial */ }
type EventTextDelta     struct { ContentIndex int; Delta string /* partial */ }
type EventTextEnd       struct { ContentIndex int; Content string /* partial */ }
type EventThinkingStart struct { ContentIndex int /* partial */ }
type EventThinkingDelta struct { ContentIndex int; Delta string /* partial */ }
type EventThinkingEnd   struct { ContentIndex int; Content string /* partial */ }
type EventToolCallStart struct { ContentIndex int /* partial */ }
type EventToolCallDelta struct { ContentIndex int; Delta string /* partial */ }
type EventToolCallEnd   struct { ContentIndex int; ToolCall ToolCall /* partial */ }
type EventDone          struct { Reason StopReason; Message AssistantMessage }
type EventError         struct { Reason StopReason; Error AssistantMessage }
```

Each carries a `partial *AssistantMessage` pointer that gets mutated in place during streaming.

---

## 4. Provider Layer (Detailed)

### 4.1 Provider Registry

```go
// registry.go
var providers sync.Map // map[string]Provider

func Register(p Provider)             { providers.Store(p.API, p) }
func Lookup(api string) (Provider, bool) { v, ok := providers.Load(api); return v.(Provider), ok }

func RegisterBuiltins() {
    Register(Provider{API: "anthropic-messages", Stream: anthropic.Stream, StreamSimple: anthropic.StreamSimple})
    Register(Provider{API: "openai-completions", Stream: openai.Stream, StreamSimple: openai.StreamSimple})
    Register(Provider{API: "zai", Stream: zai.Stream, StreamSimple: zai.StreamSimple})
}
```

### 4.2 Anthropic Provider

| | |
|---|---|
| **SDK** | `github.com/anthropics/anthropic-sdk-go` |
| **API** | Messages API with SSE streaming |

**Implementation responsibilities:**

- Convert `[]Message` → Anthropic `[]MessageParam` (handle text/image content, tool calls, tool results)
- Convert `[]Tool` → Anthropic tool definitions
- Stream SSE events → `AssistantMessageEvent` channel pushes
- Handle thinking/reasoning via `thinking` parameter with budget
- Cache control (ephemeral, 1h TTL for `api.anthropic.com`)
- Stealth mode (Claude Code tool naming compatibility)
- API key resolution: `ANTHROPIC_API_KEY` env var or explicit option
- Error mapping: HTTP errors → `EventError` with appropriate `StopReason`

### 4.3 OpenAI Provider

| | |
|---|---|
| **SDK** | `github.com/openai/openai-go` |
| **APIs** | Chat Completions (`openai-completions`), Responses (`openai-responses`) |

**Implementation responsibilities:**

- Convert `[]Message` → OpenAI `[]ChatCompletionMessageParam`
- Convert `[]Tool` → OpenAI function/tool definitions
- Parse streaming chunks: text deltas, tool call delta assembly (incremental JSON), reasoning/thinking
- Compat flags: `supportsStore`, `supportsDeveloperRole`, `maxTokensField`, `requiresThinkingAsText`
- Reasoning effort mapping: `ThinkingLevel` → `reasoning_effort` string
- Handle `stream_options: { include_usage: true }` for token counting

### 4.4 z.ai Provider

| | |
|---|---|
| **Approach** | OpenAI-compatible with z.ai extensions (`enable_thinking`, `tool_stream`) |
| **Base** | Fork the OpenAI completions provider, override z.ai-specific fields |
| **SDK** | Raw HTTP (z.ai has no official Go SDK) or reuse `openai-go` with custom base URL |

---

## 5. Agent Layer (Detailed)

### 5.1 Agent Events (`internal/agent/events.go`)

```go
type AgentEvent interface{ agentEvent() }

type AgentStartEvent struct{}                                          // agent_start
type AgentEndEvent struct{ Messages []message.Message }                // agent_end

type TurnStartEvent struct{}                                           // turn_start
type TurnEndEvent struct {                                             // turn_end
    Message     message.Message
    ToolResults []message.ToolResultMessage
}

type MessageStartEvent struct{ Message message.Message }               // message_start
type MessageUpdateEvent struct {                                       // message_update
    Message               message.Message
    AssistantMessageEvent provider.AssistantMessageEvent
}
type MessageEndEvent struct{ Message message.Message }                 // message_end

type ToolExecStartEvent struct {                                       // tool_execution_start
    ToolCallID string; ToolName string; Args map[string]any
}
type ToolExecUpdateEvent struct {                                      // tool_execution_update
    ToolCallID string; ToolName string; Args map[string]any; PartialResult any
}
type ToolExecEndEvent struct {                                         // tool_execution_end
    ToolCallID string; ToolName string; Result any; IsError bool
}
```

### 5.2 Agent Struct (`internal/agent/agent.go`)

```go
type Agent struct {
    state          *AgentState
    listeners      []func(event AgentEvent, signal context.Context)
    steeringQueue  *PendingMessageQueue
    followUpQueue  *PendingMessageQueue
    activeRun      *activeRun  // nil when idle

    // Configurable fields
    ConvertToLlm      func([]AgentMessage) ([]message.Message, error)
    TransformContext   func([]AgentMessage, context.Context) ([]AgentMessage, error)
    StreamFn          provider.StreamFn
    GetAPIKey         func(providerName string) (string, error)
    BeforeToolCall    func(BeforeToolCallContext, context.Context) (*BeforeToolCallResult, error)
    AfterToolCall     func(AfterToolCallContext, context.Context) (*AfterToolCallResult, error)
    SessionID         string
    ThinkingBudgets   provider.ThinkingBudgets
    ToolExecution     ToolExecutionMode  // "sequential" | "parallel"

    mu sync.Mutex
}

type activeRun struct {
    cancel context.CancelFunc
    done   chan struct{}
}
```

### 5.3 Core Methods

| Method | Signature |
|---|---|
| `NewAgent` | `(opts AgentOptions) *Agent` |
| `Prompt` | `(ctx, input) error` |
| `Continue` | `(ctx) error` |
| `Subscribe` | `(fn) func()` |
| `Steer` | `(msg)` |
| `FollowUp` | `(msg)` |
| `Abort` | `()` |
| `WaitForIdle` | `() <-chan struct{}` |
| `Reset` | `()` |
| `State` | `() *AgentState` |

### 5.4 Agent Loop (`internal/agent/loop.go`)

Mirrors the TS `runLoop()` exactly:

```
AgentLoop(prompts, context, config, signal)
  → emit agent_start
  → emit turn_start
  → emit message_start/end for each prompt
  → runLoop():
      while true:
        while hasToolCalls || pendingMessages:
          inject pendingMessages
          streamAssistantResponse() → emit message_start/update/end
          if error/aborted → emit agent_end, return
          executeToolCalls() → emit tool_execution_start/update/end + toolResult messages
          emit turn_end
          check steeringMessages
        check followUpMessages
        if none → break
      emit agent_end
```

`streamAssistantResponse()` — the boundary where `AgentMessage[]` becomes `Message[]`:

1. Apply `TransformContext` (optional)
2. Apply `ConvertToLlm` (required)
3. Build `provider.Context`
4. Resolve API key via `GetAPIKey`
5. Call `StreamFn` → range over event channel → emit `MessageStart/Update/End`

### 5.5 Tool Execution (`internal/agent/tool_exec.go`)

```go
type AgentTool interface {
    Name() string
    Label() string
    Description() string
    Parameters() json.RawMessage                              // JSON Schema
    PrepareArguments(raw any) (any, error)                    // Optional compat shim
    Execute(toolCallID string, params any, signal context.Context, onUpdate func(AgentToolResult)) (AgentToolResult, error)
}
```

**Parallel mode** (default):

1. Preflight all tool calls sequentially (validate args, run `beforeToolCall`)
2. Launch allowed tools concurrently via goroutines
3. Collect results in assistant source order (preserve ordering via indexed channel reads)

**Sequential mode:**

1. For each tool call: preflight → execute → finalize → next

### 5.6 State (`internal/agent/state.go`)

```go
type AgentState struct {
    SystemPrompt     string
    Model            provider.Model
    ThinkingLevel    provider.ThinkingLevel
    Tools            []AgentTool
    Messages         []AgentMessage           // AgentMessage = Message | custom (interface)
    IsStreaming       bool
    StreamingMessage  *message.Message
    PendingToolCalls  map[string]bool          // set of tool call IDs
    ErrorMessage      string
}
```

State mutations during `processEvents()` mirror the TS version: update `StreamingMessage` on start/update, clear on end, track `PendingToolCalls`, propagate errors.

---

## 6. TUI Layer (Detailed)

### 6.1 Architecture

Built with `github.com/charmbracelet/bubbletea` + `github.com/charmbracelet/lipgloss` + `github.com/charmbracelet/bubbles/textarea`.

The TUI runs a `tea.Program`. The agent runs in a background goroutine. Agent events are bridged into `tea.Msg` values via a `tea.Cmd` that reads from the agent's event channel.

### 6.2 Components

| Component | File | Description |
|---|---|---|
| `appModel` | `app.go` | Root model. Holds agent, theme, child components. Handles `tea.Update` dispatch. |
| `inputModel` | `input.go` | Multi-line text input. Enter to send, Shift+Enter for newline. |
| `chatModel` | `chat.go` | Scrollable message list. Handles viewport scrolling, auto-scroll on new content. |
| `streamingModel` | `streaming.go` | Renders in-progress assistant message with cursor animation. Accumulates deltas. |
| `toolsModel` | `tools.go` | Collapsible tool execution display. Shows tool name, args, streaming progress, result. |
| `thinkingModel` | `thinking.go` | Collapsible thinking block. Shows "Thinking..." with token count. |
| `statusModel` | `status.go` | Bottom status bar: model name, token usage, cost, streaming indicator. |
| `keymapModel` | `keymap.go` | Configurable key bindings loaded from config file / defaults. |

### 6.3 Event Bridge

```go
// messages.go
type agentEventMsg struct{ event agent.AgentEvent }
type agentDoneMsg struct{}

// In tea.Init or after prompt:
func waitForAgentEvents(ch <-chan agent.AgentEvent) tea.Cmd {
    return func() tea.Msg {
        evt, ok := <-ch
        if !ok { return agentDoneMsg{} }
        return agentEventMsg{event: evt}
    }
}

// In tea.Update:
case agentEventMsg:
    // Route to appropriate component update
    // Return waitForAgentEvents(cmd) to continue listening
```

### 6.4 Key Bindings

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Shift+Enter` | New line |
| `Ctrl+C` | Abort / quit |
| `Ctrl+O` | Toggle tool output |
| `Ctrl+L` | Toggle thinking block |
| `PgUp`/`PgDn` | Scroll chat |
| `Ctrl+R` | Reset conversation |
| `Tab` | Autocomplete |
| `Ctrl+M` | Toggle model |

---

## 7. Implementation Order

### Phase 1: Foundation (Types + Event Stream)

1. `internal/message/` — all message and content types
2. `internal/provider/events.go` — assistant message event types
3. `internal/provider/eventstream.go` — generic `EventStream[T, R]` channel-based implementation
4. `internal/provider/model.go`, `tool.go`, `options.go`, `context.go` — provider-agnostic types
5. Unit tests for event stream

### Phase 2: Provider Layer

6. `internal/provider/registry.go` — provider registration
7. `internal/provider/partial_json.go` — streaming JSON parser
8. `internal/provider/transform.go` — cross-provider message transforms
9. `internal/provider/simple_options.go` — shared option helpers
10. `internal/provider/anthropic/` — Anthropic provider (first priority, most complete reference)
11. `internal/provider/openai/` — OpenAI completions provider
12. `internal/provider/zai/` — z.ai provider
13. `internal/provider/register.go` — wire all providers
14. Integration tests with real API keys (skippable in CI)

### Phase 3: Agent Core

15. `internal/agent/events.go` — agent event types
16. `internal/agent/tool.go` — `AgentTool` interface + `AgentToolResult`
17. `internal/agent/options.go` — `AgentOptions`, `AgentLoopConfig`, hooks
18. `internal/agent/state.go` — `AgentState`
19. `internal/agent/queue.go` — `PendingMessageQueue` (steering/follow-up)
20. `internal/agent/loop.go` — `agentLoop`, `streamAssistantResponse`, `runLoop`
21. `internal/agent/tool_exec.go` — sequential + parallel tool execution
22. `internal/agent/agent.go` — stateful Agent wrapper
23. `internal/agent/proxy.go` — proxy stream function
24. Unit tests with mock provider

### Phase 4: TUI

25. `internal/tui/theme.go` — colors and styles
26. `internal/tui/keymap.go` — default key bindings
27. `internal/tui/input.go` — text input component
28. `internal/tui/chat.go` — message list + viewport
29. `internal/tui/streaming.go` — live text rendering
30. `internal/tui/thinking.go` — thinking block display
31. `internal/tui/tools.go` — tool execution display
32. `internal/tui/status.go` — status bar
33. `internal/tui/messages.go` — event bridge (agent events → `tea.Msg`)
34. `internal/tui/app.go` — root model wiring
35. `main.go` — entrypoint

---

## 8. Key Go Design Decisions

| TS Pattern | Go Equivalent |
|---|---|
| `AsyncIterable<T>` | `<-chan T` (read-only channel) |
| `Promise<R>` | `<-chan R` or `sync.WaitGroup` |
| `subscribe(listener)` | `Subscribe(func(AgentEvent, context.Context))` |
| `abortController.signal` | `context.WithCancel` |
| Declaration merging (custom messages) | `AgentMessage` interface with `messageUnion()` |
| `class EventStream` with push/end | `EventStream[T,R]` struct with `Push/End` + goroutine bridge |
| `typebox` schema for tool params | `json.RawMessage` (raw JSON Schema) |
| `Array.slice()` defensive copy | Return slice copy in getter, copy in setter |
| `await listener(event, signal)` | Call listener synchronously in order |
| Lazy module loading | `sync.Once` or init-time registration |

---

## 9. Testing Strategy

- **Unit tests:** Agent loop with mock stream function (returns canned events)
- **Provider tests:** Table-driven tests for message conversion, streaming JSON parsing
- **Integration tests:** Real API calls behind build tags (`//go:build integration`)
- **TUI tests:** Golden file tests for rendering snapshots

---

## 10. Dependencies

| Package | Purpose |
|---|---|
| `github.com/anthropics/anthropic-sdk-go` | Anthropic provider |
| `github.com/openai/openai-go` | OpenAI provider |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components (textarea, viewport) |
| `github.com/xeipuuv/gojsonschema` | JSON Schema validation for tool args |
| `github.com/stretchr/testify` | Test assertions |
