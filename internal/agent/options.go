package agent

import (
	"context"

	"github.com/fantods/yaah/internal/provider"
)

type BeforeToolCallHook func(ctx context.Context, btx BeforeToolCallContext) (*AgentToolResult, error)

type AfterToolCallHook func(ctx context.Context, atx AfterToolCallContext)

type AgentLoopConfig struct {
	MaxTurns          int  `json:"maxTurns"`
	ParallelToolCalls bool `json:"parallelToolCalls"`
}

type BeforeToolCallContext struct {
	ToolName string         `json:"toolName"`
	ToolCall AgentToolCall  `json:"toolCall"`
	Args     map[string]any `json:"args"`
}

type AfterToolCallContext struct {
	ToolName string           `json:"toolName"`
	ToolCall AgentToolCall    `json:"toolCall"`
	Result   *AgentToolResult `json:"result"`
}

type AgentOptions struct {
	Model        provider.Model         `json:"model"`
	SystemPrompt string                 `json:"systemPrompt,omitempty"`
	Tools        []AgentTool            `json:"-"`
	LoopConfig   AgentLoopConfig        `json:"loopConfig"`
	StreamOpts   provider.StreamOptions `json:"streamOpts"`

	BeforeToolCall BeforeToolCallHook `json:"-"`
	AfterToolCall  AfterToolCallHook  `json:"-"`
}
