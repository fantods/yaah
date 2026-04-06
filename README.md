# yaah - Yet Another Agent Harness

Stateful coding agent in Go with tool execution, event streaming via typed channels, multi-provider support, and a bubbletea TUI. 

## Supported Providers

- **Anthropic** (Messages API, SSE streaming)
- **OpenAI** (Chat Completions + Responses API)
- **z.ai** (OpenAI-compatible with extensions)

## Quick Start

Requires [Go 1.21+](https://go.dev/dl/).

### 1. Install

```bash
go install github.com/fantods/yaah@latest
```

This places a `yaah` binary in `$GOPATH/bin` (or `$HOME/go/bin` by default). Make sure that directory is on your `PATH`.

### 2. Set an API key

At least one provider key is required. Export the environment variable for your chosen provider:

| Provider | Variable | Get a key |
|----------|----------|-----------|
| Anthropic | `ANTHROPIC_API_KEY` | https://console.anthropic.com |
| OpenAI | `OPENAI_API_KEY` | https://platform.openai.com |
| z.ai | `ZAI_API_KEY` | https://z.ai |

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

### 3. Run

```bash
yaah
```

Providers (Anthropic, OpenAI, z.ai) register themselves via `init()` — no manual registration needed. Just import the ones you need with blank imports if building your own entrypoint:

```go
import (
    _ "github.com/fantods/yaah/internal/provider/anthropic"
    _ "github.com/fantods/yaah/internal/provider/openai"
    _ "github.com/fantods/yaah/internal/provider/zai"
)
```

Type a message and press Enter. The agent streams the response in real-time. Press `?` for keybindings.

### 4. (Optional) Configure the agent

**System prompt** — set a system prompt to control the agent's behavior:

```go
a := agent.NewAgent(
    agent.AgentOptions{
        Model:        model,
        SystemPrompt: "You are a helpful coding assistant.",
        LoopConfig: agent.AgentLoopConfig{
            MaxTurns: 10,
        },
    },
    agent.StreamProxy,
)
```

**Tools** — register tools the agent can call during conversation:

```go
readFileTool := agent.AgentTool{
    Name:        "read_file",
    Label:       "Read File",
    Description: "Read a file's contents",
    Parameters:  schema,
    Execute: func(toolCallID string, params any, signal context.Context, onUpdate func(agent.AgentToolResult)) (agent.AgentToolResult, error) {
        // ... execution logic
    },
}

a := agent.NewAgent(
    agent.AgentOptions{
        Model: model,
        Tools: []agent.AgentTool{readFileTool},
        // ...
    },
    agent.StreamProxy,
)
```

**Thinking level** — enable extended thinking for supported models:

```go
agent.AgentOptions{
    // ...
    StreamOpts: provider.StreamOptions{
        ThinkingLevel: provider.ThinkingLevelMedium,
    },
}
```

## Architecture

Three layers, each independently testable:

```
internal/message   — Core message types shared across all packages
internal/provider  — Provider abstraction, event stream, Anthropic/OpenAI/z.ai
internal/agent     — Stateful agent loop with tool execution
internal/tui       — Bubbletea terminal interface
```

### Message Flow

```
AgentMessage[] → transformContext() → AgentMessage[] → convertToLlm() → Message[] → LLM
                    (optional)                           (required)
```

### Event Flow

The agent emits typed events over Go channels. Consumers range over the event channel:

```
prompt("Hello")
├─ agent_start
├─ turn_start
├─ message_start   { userMessage }
├─ message_end     { userMessage }
├─ message_start   { assistantMessage }
├─ message_update  { partial... }       // streaming chunks
├─ message_end     { assistantMessage }
├─ turn_end
└─ agent_end
```

With tool calls, the loop continues until no more tool calls are pending:

```
├─ message_end     { assistantMessage with toolCall }
├─ tool_execution_start  { toolName, args }
├─ tool_execution_end    { result }
├─ message_start/end { toolResultMessage }
├─ turn_end
├─ turn_start                              // next turn
├─ message_start { assistantMessage }      // LLM responds to tool result
├─ message_end
└─ agent_end
```

### Event Stream

The `EventStream[T, R]` generic type is the Go equivalent of the TS `EventStream` class:

- `Events()` returns a `<-chan T` for consumers to range over
- `Result()` returns a `<-chan R` that resolves when the stream completes
- `Push()` sends events non-blocking with overflow protection

### Tool Execution

Two modes, configurable per agent:

- **parallel** (default): preflight sequentially, execute concurrently, collect results in source order
- **sequential**: execute one at a time

## Agent Options

```go
agent := agent.NewAgent(agent.AgentOptions{
    State: &agent.AgentState{
        SystemPrompt:  "You are helpful.",
        Model:         model,
        ThinkingLevel: provider.ThinkingLevelMedium,
        Tools:         []agent.AgentTool{readFileTool, bashTool},
    },

    ConvertToLlm:    func(msgs []agent.AgentMessage) ([]message.Message, error) { ... },
    TransformContext: func(msgs []agent.AgentMessage, ctx context.Context) ([]agent.AgentMessage, error) { ... },
    StreamFn:        provider.StreamFn,
    GetAPIKey:       func(providerName string) (string, error) { ... },
    BeforeToolCall:  func(ctx agent.BeforeToolCallContext, cancel context.Context) (*agent.BeforeToolCallResult, error) { ... },
    AfterToolCall:   func(ctx agent.AfterToolCallContext, cancel context.Context) (*agent.AfterToolCallResult, error) { ... },
    ToolExecution:   agent.ToolExecutionParallel,
})
```

## Tools

```go
readFileTool := agent.AgentTool{
    Name:        "read_file",
    Label:       "Read File",
    Description: "Read a file's contents",
    Parameters:  schema,
    Execute: func(toolCallID string, params any, signal context.Context, onUpdate func(agent.AgentToolResult)) (agent.AgentToolResult, error) {
        path := params.(map[string]any)["path"].(string)
        data, err := os.ReadFile(path)
        if err != nil {
            return agent.AgentToolResult{}, err
        }
        return agent.AgentToolResult{
            Content: []message.ContentBlock{
                message.TextContent{Type: "text", Text: string(data)},
            },
        }, nil
    },
}
```

## Steering and Follow-up

Steering messages interrupt the agent mid-loop. Follow-up messages queue work after completion.

```go
agent.Steer(userMessage)   // inject while tools are running
agent.FollowUp(userMessage) // queue for after current work finishes
agent.Abort()               // cancel current operation
```

## Environment Variables

| Provider | Variable |
|----------|----------|
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| z.ai | `ZAI_API_KEY` |

## License

MIT
