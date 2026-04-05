package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/assert"
)

type MockTool struct {
	ToolInfo provider.Tool
	Result   *AgentToolResult
	Err      error
}

func (m *MockTool) Info() provider.Tool {
	return m.ToolInfo
}

func (m *MockTool) Run(_ context.Context, _ AgentToolCall) (*AgentToolResult, error) {
	return m.Result, m.Err
}

func TestPhase0Smoke(t *testing.T) {
	t.Log("agent package compiles")
}

func TestAgentStartEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = AgentStartEvent{}
}

func TestAgentEndEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = AgentEndEvent{
		Messages: []message.Message{},
	}
}

func TestTurnStartEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = TurnStartEvent{}
}

func TestTurnEndEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = TurnEndEvent{
		Message:     message.AssistantMessage{},
		ToolResults: []message.ToolResultMessage{},
	}
}

func TestMessageStartEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = MessageStartEvent{
		Message: message.AssistantMessage{},
	}
}

func TestMessageUpdateEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = MessageUpdateEvent{
		Message:               message.AssistantMessage{},
		AssistantMessageEvent: provider.EventTextDelta{},
	}
}

func TestMessageEndEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = MessageEndEvent{
		Message: message.AssistantMessage{},
	}
}

func TestToolExecStartEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = ToolExecStartEvent{
		ToolCallID: "call_1",
		ToolName:   "bash",
		Args:       map[string]any{"cmd": "ls"},
	}
}

func TestToolExecUpdateEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = ToolExecUpdateEvent{
		ToolCallID:    "call_1",
		ToolName:      "bash",
		Args:          map[string]any{"cmd": "ls"},
		PartialResult: "file.txt\n",
	}
}

func TestToolExecEndEventImplementsAgentEvent(t *testing.T) {
	var _ AgentEvent = ToolExecEndEvent{
		ToolCallID: "call_1",
		ToolName:   "bash",
		Result:     "file.txt\n",
		IsError:    false,
	}
}

func TestAgentEventSwitch(t *testing.T) {
	events := []AgentEvent{
		AgentStartEvent{},
		TurnStartEvent{},
		MessageStartEvent{Message: message.UserMessage{}},
		MessageUpdateEvent{
			Message:               message.AssistantMessage{},
			AssistantMessageEvent: provider.EventTextDelta{Delta: "hi"},
		},
		MessageEndEvent{Message: message.AssistantMessage{}},
		ToolExecStartEvent{ToolCallID: "c1", ToolName: "bash", Args: map[string]any{}},
		ToolExecUpdateEvent{ToolCallID: "c1", ToolName: "bash", PartialResult: "out"},
		ToolExecEndEvent{ToolCallID: "c1", ToolName: "bash", Result: "out", IsError: false},
		TurnEndEvent{
			Message:     message.AssistantMessage{},
			ToolResults: []message.ToolResultMessage{},
		},
		AgentEndEvent{Messages: []message.Message{}},
	}

	for _, evt := range events {
		switch evt.(type) {
		case AgentStartEvent:
		case AgentEndEvent:
		case TurnStartEvent:
		case TurnEndEvent:
		case MessageStartEvent:
		case MessageUpdateEvent:
		case MessageEndEvent:
		case ToolExecStartEvent:
		case ToolExecUpdateEvent:
		case ToolExecEndEvent:
		default:
			t.Fatalf("unexpected event type: %T", evt)
		}
	}

	assert.Equal(t, 10, len(events))
}

func TestToolExecEndEventIsError(t *testing.T) {
	errEvt := ToolExecEndEvent{
		ToolCallID: "c1",
		ToolName:   "bash",
		Result:     "command failed",
		IsError:    true,
	}
	assert.True(t, errEvt.IsError)

	okEvt := ToolExecEndEvent{
		ToolCallID: "c1",
		ToolName:   "bash",
		Result:     "ok",
		IsError:    false,
	}
	assert.False(t, okEvt.IsError)
}

func TestMockToolImplementsAgentTool(t *testing.T) {
	tool := &MockTool{
		ToolInfo: provider.Tool{
			Name:        "mock",
			Description: "a mock tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{},
			IsError: false,
		},
	}
	var _ AgentTool = tool
}

func TestAgentToolResultConstruction(t *testing.T) {
	result := &AgentToolResult{
		Content: []message.ContentBlock{
			message.TextContent{Text: "hello"},
		},
		IsError: false,
	}
	assert.Equal(t, 1, len(result.Content))
	assert.False(t, result.IsError)
}

func TestAgentToolResultError(t *testing.T) {
	result := &AgentToolResult{
		Content: []message.ContentBlock{
			message.TextContent{Text: "something went wrong"},
		},
		IsError: true,
	}
	assert.True(t, result.IsError)
}

func TestAgentToolCallFields(t *testing.T) {
	call := AgentToolCall{
		ID:   "call_123",
		Name: "bash",
		Args: map[string]any{"cmd": "ls"},
	}
	assert.Equal(t, "call_123", call.ID)
	assert.Equal(t, "bash", call.Name)
	assert.Equal(t, map[string]any{"cmd": "ls"}, call.Args)
}

func TestMockToolRunReturnsResult(t *testing.T) {
	expected := &AgentToolResult{
		Content: []message.ContentBlock{
			message.TextContent{Text: "output"},
		},
		IsError: false,
	}
	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "echo"},
		Result:   expected,
	}
	call := AgentToolCall{
		ID:   "c1",
		Name: "echo",
		Args: map[string]any{},
	}
	got, err := tool.Run(context.Background(), call)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestMockToolRunReturnsError(t *testing.T) {
	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "fail"},
		Err:      errors.New("boom"),
	}
	call := AgentToolCall{
		ID:   "c1",
		Name: "fail",
		Args: map[string]any{},
	}
	_, err := tool.Run(context.Background(), call)
	assert.Error(t, err)
	assert.Equal(t, "boom", err.Error())
}

func TestAgentOptionsConstruction(t *testing.T) {
	opts := AgentOptions{
		Model: provider.Model{
			ID:  "gpt-4",
			API: "openai",
		},
		SystemPrompt: "You are helpful.",
		Tools:        []AgentTool{},
		LoopConfig: AgentLoopConfig{
			MaxTurns:          10,
			ParallelToolCalls: true,
		},
		StreamOpts: provider.StreamOptions{},
	}
	assert.Equal(t, "gpt-4", opts.Model.ID)
	assert.Equal(t, "You are helpful.", opts.SystemPrompt)
	assert.Equal(t, 10, opts.LoopConfig.MaxTurns)
	assert.True(t, opts.LoopConfig.ParallelToolCalls)
}

func TestAgentOptionsWithHooks(t *testing.T) {
	var beforeCalled bool
	var afterCalled bool

	opts := AgentOptions{
		BeforeToolCall: func(_ context.Context, _ BeforeToolCallContext) (*AgentToolResult, error) {
			beforeCalled = true
			return nil, nil
		},
		AfterToolCall: func(_ context.Context, _ AfterToolCallContext) {
			afterCalled = true
		},
	}

	result, err := opts.BeforeToolCall(context.Background(), BeforeToolCallContext{})
	assert.Nil(t, result)
	assert.NoError(t, err)
	assert.True(t, beforeCalled)

	opts.AfterToolCall(context.Background(), AfterToolCallContext{})
	assert.True(t, afterCalled)
}

func TestBeforeToolCallContext(t *testing.T) {
	btx := BeforeToolCallContext{
		ToolName: "bash",
		ToolCall: AgentToolCall{
			ID:   "c1",
			Name: "bash",
			Args: map[string]any{"cmd": "ls"},
		},
		Args: map[string]any{"cmd": "ls"},
	}
	assert.Equal(t, "bash", btx.ToolName)
	assert.Equal(t, "c1", btx.ToolCall.ID)
	assert.Equal(t, "ls", btx.Args["cmd"])
}

func TestAfterToolCallContext(t *testing.T) {
	atx := AfterToolCallContext{
		ToolName: "bash",
		ToolCall: AgentToolCall{
			ID:   "c1",
			Name: "bash",
			Args: map[string]any{},
		},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{
				message.TextContent{Text: "output"},
			},
			IsError: false,
		},
	}
	assert.Equal(t, "bash", atx.ToolName)
	assert.Equal(t, "c1", atx.ToolCall.ID)
	assert.Equal(t, "output", atx.Result.Content[0].(message.TextContent).Text)
	assert.False(t, atx.Result.IsError)
}

func TestBeforeToolCallHookCanShortCircuit(t *testing.T) {
	shortCircuitResult := &AgentToolResult{
		Content: []message.ContentBlock{
			message.TextContent{Text: "blocked"},
		},
		IsError: true,
	}

	opts := AgentOptions{
		BeforeToolCall: func(_ context.Context, _ BeforeToolCallContext) (*AgentToolResult, error) {
			return shortCircuitResult, nil
		},
	}

	got, err := opts.BeforeToolCall(context.Background(), BeforeToolCallContext{
		ToolName: "dangerous",
	})
	assert.NoError(t, err)
	assert.Equal(t, shortCircuitResult, got)
	assert.True(t, got.IsError)
}

func TestAgentLoopConfigDefaults(t *testing.T) {
	cfg := AgentLoopConfig{}
	assert.Equal(t, 0, cfg.MaxTurns)
	assert.False(t, cfg.ParallelToolCalls)
}
