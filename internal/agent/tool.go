package agent

import (
	"context"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

type AgentTool interface {
	Info() provider.Tool
	Run(ctx context.Context, call AgentToolCall) (*AgentToolResult, error)
}

type AgentToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type AgentToolResult struct {
	Content []message.ContentBlock `json:"content"`
	IsError bool                   `json:"isError"`
}
