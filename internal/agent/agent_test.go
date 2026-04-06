package agent

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
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

func TestNewAgentState(t *testing.T) {
	s := NewAgentState()
	assert.False(t, s.IsStreaming())
	assert.Equal(t, []message.Message{}, s.GetMessages())
	assert.Equal(t, []AgentToolCall{}, s.GetPendingToolCalls())
	assert.Nil(t, s.GetError())
	assert.Equal(t, 0, s.GetTurn())
}

func TestAgentStateSetStreaming(t *testing.T) {
	s := NewAgentState()
	s.SetStreaming(true)
	assert.True(t, s.IsStreaming())
	s.SetStreaming(false)
	assert.False(t, s.IsStreaming())
}

func TestAgentStateAddMessage(t *testing.T) {
	s := NewAgentState()
	s.AddMessage(message.UserMessage{Role: "user", Content: []message.ContentBlock{}})
	s.AddMessage(message.AssistantMessage{Role: "assistant", Content: []message.ContentBlock{}})

	msgs := s.GetMessages()
	assert.Equal(t, 2, len(msgs))
	assert.IsType(t, message.UserMessage{}, msgs[0])
	assert.IsType(t, message.AssistantMessage{}, msgs[1])
}

func TestAgentStateGetMessagesReturnsCopy(t *testing.T) {
	s := NewAgentState()
	s.AddMessage(message.UserMessage{Role: "user"})
	original := s.GetMessages()
	s.AddMessage(message.AssistantMessage{Role: "assistant"})

	assert.Equal(t, 1, len(original))
	assert.Equal(t, 2, len(s.GetMessages()))
}

func TestAgentStatePendingToolCalls(t *testing.T) {
	s := NewAgentState()
	calls := []AgentToolCall{
		{ID: "c1", Name: "bash", Args: map[string]any{"cmd": "ls"}},
		{ID: "c2", Name: "read", Args: map[string]any{"file": "x.go"}},
	}
	s.SetPendingToolCalls(calls)

	got := s.GetPendingToolCalls()
	assert.Equal(t, 2, len(got))
	assert.Equal(t, "c1", got[0].ID)
	assert.Equal(t, "c2", got[1].ID)
}

func TestAgentStateGetPendingToolCallsReturnsCopy(t *testing.T) {
	s := NewAgentState()
	s.SetPendingToolCalls([]AgentToolCall{{ID: "c1", Name: "bash"}})
	got := s.GetPendingToolCalls()
	got[0].ID = "mutated"

	assert.Equal(t, "c1", s.GetPendingToolCalls()[0].ID)
}

func TestAgentStateError(t *testing.T) {
	s := NewAgentState()
	assert.Nil(t, s.GetError())

	s.SetError(errors.New("something broke"))
	assert.Equal(t, "something broke", s.GetError().Error())

	s.SetError(nil)
	assert.Nil(t, s.GetError())
}

func TestAgentStateTurn(t *testing.T) {
	s := NewAgentState()
	assert.Equal(t, 0, s.GetTurn())

	s.IncrementTurn()
	assert.Equal(t, 1, s.GetTurn())

	s.IncrementTurn()
	s.IncrementTurn()
	assert.Equal(t, 3, s.GetTurn())
}

func TestNewPendingMessageQueue(t *testing.T) {
	q := NewPendingMessageQueue()
	assert.Equal(t, 0, q.Len())
}

func TestQueueEnqueueDequeue(t *testing.T) {
	q := NewPendingMessageQueue()
	msg := message.UserMessage{Role: "user", Content: []message.ContentBlock{}}
	q.Enqueue(QueueModeFollowUp, msg)

	mode, got, ok := q.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, QueueModeFollowUp, mode)
	assert.Equal(t, msg, got)
	assert.Equal(t, 0, q.Len())
}

func TestQueueDequeueEmpty(t *testing.T) {
	q := NewPendingMessageQueue()
	_, _, ok := q.Dequeue()
	assert.False(t, ok)
}

func TestQueueFIFO(t *testing.T) {
	q := NewPendingMessageQueue()
	first := message.UserMessage{Role: "user", Timestamp: 1}
	second := message.UserMessage{Role: "user", Timestamp: 2}
	q.Enqueue(QueueModeFollowUp, first)
	q.Enqueue(QueueModeFollowUp, second)

	_, got1, _ := q.Dequeue()
	assert.Equal(t, int64(1), got1.(message.UserMessage).Timestamp)

	_, got2, _ := q.Dequeue()
	assert.Equal(t, int64(2), got2.(message.UserMessage).Timestamp)
}

func TestQueueSteeringVsFollowUp(t *testing.T) {
	q := NewPendingMessageQueue()
	followUp := message.UserMessage{Role: "user", Timestamp: 1}
	steering := message.UserMessage{Role: "user", Timestamp: 2}
	q.Enqueue(QueueModeFollowUp, followUp)
	q.Enqueue(QueueModeSteering, steering)

	mode, _, _ := q.Dequeue()
	assert.Equal(t, QueueModeFollowUp, mode)

	mode, _, _ = q.Dequeue()
	assert.Equal(t, QueueModeSteering, mode)
}

func TestQueueDequeueByMode(t *testing.T) {
	q := NewPendingMessageQueue()
	followUp := message.UserMessage{Role: "user", Timestamp: 1}
	steering := message.UserMessage{Role: "user", Timestamp: 2}
	q.Enqueue(QueueModeFollowUp, followUp)
	q.Enqueue(QueueModeSteering, steering)

	got, ok := q.DequeueByMode(QueueModeSteering)
	assert.True(t, ok)
	assert.Equal(t, int64(2), got.(message.UserMessage).Timestamp)
	assert.Equal(t, 1, q.Len())

	_, ok = q.DequeueByMode(QueueModeSteering)
	assert.False(t, ok)
}

func TestQueueDequeueByModeFollowUp(t *testing.T) {
	q := NewPendingMessageQueue()
	first := message.UserMessage{Role: "user", Timestamp: 1}
	second := message.UserMessage{Role: "user", Timestamp: 2}
	q.Enqueue(QueueModeFollowUp, first)
	q.Enqueue(QueueModeSteering, second)

	got, ok := q.DequeueByMode(QueueModeFollowUp)
	assert.True(t, ok)
	assert.Equal(t, int64(1), got.(message.UserMessage).Timestamp)
	assert.Equal(t, 1, q.Len())
}

func TestQueueClear(t *testing.T) {
	q := NewPendingMessageQueue()
	q.Enqueue(QueueModeFollowUp, message.UserMessage{})
	q.Enqueue(QueueModeSteering, message.UserMessage{})
	assert.Equal(t, 2, q.Len())

	q.Clear()
	assert.Equal(t, 0, q.Len())
}

func mockTextStream(msg message.AssistantMessage, deltas []string) *provider.AssistantMessageEventStream {
	stream := provider.NewEventStream(
		func(provider.AssistantMessageEvent) bool { return false },
		func(provider.AssistantMessageEvent) message.AssistantMessage { return msg },
	)

	go func() {
		stream.Push(provider.EventStart{Partial: &msg})
		for i, delta := range deltas {
			stream.Push(provider.EventTextStart{
				Partial:      &msg,
				ContentIndex: i,
			})
			stream.Push(provider.EventTextDelta{
				Partial:      &msg,
				ContentIndex: i,
				Delta:        delta,
			})
			stream.Push(provider.EventTextEnd{
				Partial:      &msg,
				ContentIndex: i,
				Content:      delta,
			})
		}
		stream.Push(provider.EventDone{
			Reason:  message.StopReasonStop,
			Message: msg,
		})
		stream.End(&msg)
	}()

	return stream
}

func collectEvents(ch <-chan AgentEvent) []AgentEvent {
	events := []AgentEvent{}
	for evt := range ch {
		events = append(events, evt)
	}
	return events
}

func TestAgentLoopTextOnlyTurn(t *testing.T) {
	msg := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "Hello world"}},
		StopReason: message.StopReasonStop,
	}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		return mockTextStream(msg, []string{"Hello", " world"})
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	expected := []struct {
		typ string
	}{
		{"AgentStartEvent"},
		{"TurnStartEvent"},
		{"MessageStartEvent"},
		{"MessageUpdateEvent"},
		{"MessageUpdateEvent"},
		{"MessageEndEvent"},
		{"TurnEndEvent"},
		{"AgentEndEvent"},
	}

	assert.Equal(t, len(expected), len(events), "event count mismatch")

	for i, exp := range expected {
		switch exp.typ {
		case "AgentStartEvent":
			assert.IsType(t, AgentStartEvent{}, events[i])
		case "TurnStartEvent":
			assert.IsType(t, TurnStartEvent{}, events[i])
		case "MessageStartEvent":
			assert.IsType(t, MessageStartEvent{}, events[i])
		case "MessageUpdateEvent":
			assert.IsType(t, MessageUpdateEvent{}, events[i])
		case "MessageEndEvent":
			assert.IsType(t, MessageEndEvent{}, events[i])
		case "TurnEndEvent":
			te := events[i].(TurnEndEvent)
			assert.IsType(t, message.AssistantMessage{}, te.Message)
			assert.Equal(t, []message.ToolResultMessage{}, te.ToolResults)
		case "AgentEndEvent":
			ae := events[i].(AgentEndEvent)
			assert.Equal(t, 2, len(ae.Messages))
		}
	}
}

func TestAgentLoopStateUpdatedAfterTextTurn(t *testing.T) {
	msg := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "hi"}},
		StopReason: message.StopReasonStop,
	}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		return mockTextStream(msg, []string{"hi"})
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}

	collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	assert.Equal(t, 2, len(state.GetMessages()))
	assert.Equal(t, 1, state.GetTurn())
	assert.False(t, state.IsStreaming())
	assert.Nil(t, state.GetError())
}

func TestAgentLoopBuildsProviderContext(t *testing.T) {
	msg := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "ok"}},
		StopReason: message.StopReasonStop,
	}

	var capturedCtx provider.Context
	streamFn := func(_ provider.Model, ctx provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		capturedCtx = ctx
		return mockTextStream(msg, []string{"ok"})
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user", Content: []message.ContentBlock{}})

	opts := AgentOptions{
		Model:        provider.Model{ID: "test", API: "test"},
		SystemPrompt: "be helpful",
		Tools: []AgentTool{
			&MockTool{ToolInfo: provider.Tool{Name: "bash", Description: "run commands"}},
		},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}

	collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	assert.Equal(t, "be helpful", capturedCtx.SystemPrompt)
	assert.Equal(t, 1, len(capturedCtx.Messages))
	assert.Equal(t, 1, len(capturedCtx.Tools))
	assert.Equal(t, "bash", capturedCtx.Tools[0].Name)
}

func TestAgentLoopEmptyDeltaStream(t *testing.T) {
	msg := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{},
		StopReason: message.StopReasonStop,
	}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		return mockTextStream(msg, []string{})
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	hasMessageStart := false
	hasMessageEnd := false
	for _, evt := range events {
		switch evt.(type) {
		case MessageStartEvent:
			hasMessageStart = true
		case MessageEndEvent:
			hasMessageEnd = true
		}
	}
	assert.True(t, hasMessageStart)
	assert.True(t, hasMessageEnd)
}

func mockToolCallStream(msg message.AssistantMessage, toolCalls []message.ToolCall) *provider.AssistantMessageEventStream {
	stream := provider.NewEventStream(
		func(provider.AssistantMessageEvent) bool { return false },
		func(provider.AssistantMessageEvent) message.AssistantMessage { return msg },
	)

	go func() {
		stream.Push(provider.EventStart{Partial: &msg})
		for i, tc := range toolCalls {
			stream.Push(provider.EventToolCallStart{
				Partial:      &msg,
				ContentIndex: i,
			})
			stream.Push(provider.EventToolCallEnd{
				Partial:      &msg,
				ContentIndex: i,
				ToolCall:     tc,
			})
		}
		stream.Push(provider.EventDone{
			Reason:  message.StopReasonToolUse,
			Message: msg,
		})
		stream.End(&msg)
	}()

	return stream
}

func mockMultiTurnStreamFn(responses []message.AssistantMessage, toolCallsPerTurn [][]message.ToolCall) provider.StreamFn {
	callCount := 0
	return func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		idx := callCount
		callCount++
		if idx < len(toolCallsPerTurn) && len(toolCallsPerTurn[idx]) > 0 {
			return mockToolCallStream(responses[idx], toolCallsPerTurn[idx])
		}
		return mockTextStream(responses[idx], []string{extractText(responses[idx])})
	}
}

func extractText(msg message.AssistantMessage) string {
	for _, block := range msg.Content {
		if tc, ok := block.(message.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestAgentLoopWithSingleToolCall(t *testing.T) {
	assistantWithTool := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.TextContent{Text: "let me check"},
			message.ToolCall{
				Type:      "toolCall",
				ID:        "call_1",
				Name:      "bash",
				Arguments: map[string]any{"cmd": "ls"},
			},
		},
		StopReason: message.StopReasonToolUse,
	}

	finalResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "here are the files"}},
		StopReason: message.StopReasonStop,
	}

	toolCalls := []message.ToolCall{
		{Type: "toolCall", ID: "call_1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}},
	}

	callCount := 0
	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		defer func() { callCount++ }()
		if callCount == 0 {
			return mockToolCallStream(assistantWithTool, toolCalls)
		}
		return mockTextStream(finalResponse, []string{"here are the files"})
	}

	mockTool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash", Description: "run commands"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "file.txt\n"}},
			IsError: false,
		},
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		Tools:      []AgentTool{mockTool},
		LoopConfig: AgentLoopConfig{MaxTurns: 5},
	}

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	var turnStarts, turnEnds, toolExecStarts, toolExecEnds int
	for _, evt := range events {
		switch evt.(type) {
		case TurnStartEvent:
			turnStarts++
		case TurnEndEvent:
			turnEnds++
		case ToolExecStartEvent:
			toolExecStarts++
		case ToolExecEndEvent:
			toolExecEnds++
		}
	}

	assert.Equal(t, 2, turnStarts, "should have 2 turns")
	assert.Equal(t, 2, turnEnds, "should have 2 turn ends")
	assert.Equal(t, 1, toolExecStarts, "should have 1 tool exec start")
	assert.Equal(t, 1, toolExecEnds, "should have 1 tool exec end")
	assert.Equal(t, 4, len(state.GetMessages()), "user + assistant + tool result + final assistant")
	assert.Nil(t, state.GetError())
}

func TestAgentLoopWithMultipleToolCalls(t *testing.T) {
	assistantWithTools := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.ToolCall{
				Type: "toolCall", ID: "call_1", Name: "bash",
				Arguments: map[string]any{"cmd": "ls"},
			},
			message.ToolCall{
				Type: "toolCall", ID: "call_2", Name: "read",
				Arguments: map[string]any{"file": "x.go"},
			},
		},
		StopReason: message.StopReasonToolUse,
	}

	finalResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "done"}},
		StopReason: message.StopReasonStop,
	}

	toolCalls := []message.ToolCall{
		{Type: "toolCall", ID: "call_1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}},
		{Type: "toolCall", ID: "call_2", Name: "read", Arguments: map[string]any{"file": "x.go"}},
	}

	callCount := 0
	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		defer func() { callCount++ }()
		if callCount == 0 {
			return mockToolCallStream(assistantWithTools, toolCalls)
		}
		return mockTextStream(finalResponse, []string{"done"})
	}

	bashTool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "file.txt"}},
			IsError: false,
		},
	}
	readTool := &MockTool{
		ToolInfo: provider.Tool{Name: "read"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "package main"}},
			IsError: false,
		},
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		Tools:      []AgentTool{bashTool, readTool},
		LoopConfig: AgentLoopConfig{MaxTurns: 5},
	}

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	var toolExecStarts, toolExecEnds int
	for _, evt := range events {
		switch e := evt.(type) {
		case ToolExecStartEvent:
			toolExecStarts++
			assert.Contains(t, []string{"call_1", "call_2"}, e.ToolCallID)
		case ToolExecEndEvent:
			toolExecEnds++
			assert.False(t, e.IsError)
		}
	}

	assert.Equal(t, 2, toolExecStarts)
	assert.Equal(t, 2, toolExecEnds)
	assert.Equal(t, 5, len(state.GetMessages()), "user + assistant + 2 tool results + final assistant")
}

func TestAgentLoopToolNotFound(t *testing.T) {
	assistantWithTool := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.ToolCall{
				Type: "toolCall", ID: "call_1", Name: "missing",
				Arguments: map[string]any{},
			},
		},
		StopReason: message.StopReasonToolUse,
	}

	finalResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "ok"}},
		StopReason: message.StopReasonStop,
	}

	callCount := 0
	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		defer func() { callCount++ }()
		if callCount == 0 {
			return mockToolCallStream(assistantWithTool, []message.ToolCall{
				{Type: "toolCall", ID: "call_1", Name: "missing", Arguments: map[string]any{}},
			})
		}
		return mockTextStream(finalResponse, []string{"ok"})
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		Tools:      []AgentTool{},
		LoopConfig: AgentLoopConfig{MaxTurns: 5},
	}

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	var toolExecEnd ToolExecEndEvent
	for _, evt := range events {
		if e, ok := evt.(ToolExecEndEvent); ok {
			toolExecEnd = e
			break
		}
	}
	assert.True(t, toolExecEnd.IsError)
	assert.Equal(t, "tool not found: missing", toolExecEnd.Result)
}

func TestAgentLoopToolRunError(t *testing.T) {
	assistantWithTool := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.ToolCall{
				Type: "toolCall", ID: "call_1", Name: "fail",
				Arguments: map[string]any{},
			},
		},
		StopReason: message.StopReasonToolUse,
	}

	finalResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "sorry"}},
		StopReason: message.StopReasonStop,
	}

	callCount := 0
	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		defer func() { callCount++ }()
		if callCount == 0 {
			return mockToolCallStream(assistantWithTool, []message.ToolCall{
				{Type: "toolCall", ID: "call_1", Name: "fail", Arguments: map[string]any{}},
			})
		}
		return mockTextStream(finalResponse, []string{"sorry"})
	}

	failTool := &MockTool{
		ToolInfo: provider.Tool{Name: "fail"},
		Err:      errors.New("execution failed"),
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		Tools:      []AgentTool{failTool},
		LoopConfig: AgentLoopConfig{MaxTurns: 5},
	}

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	var toolExecEnd ToolExecEndEvent
	for _, evt := range events {
		if e, ok := evt.(ToolExecEndEvent); ok {
			toolExecEnd = e
		}
	}
	assert.True(t, toolExecEnd.IsError)
	assert.Equal(t, "execution failed", toolExecEnd.Result)
}

func TestAgentLoopRespectsMaxTurns(t *testing.T) {
	toolUseResponse := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.ToolCall{
				Type: "toolCall", ID: "call_1", Name: "bash",
				Arguments: map[string]any{"cmd": "ls"},
			},
		},
		StopReason: message.StopReasonToolUse,
	}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		return mockToolCallStream(toolUseResponse, []message.ToolCall{
			{Type: "toolCall", ID: "call_1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}},
		})
	}

	bashTool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "out"}},
			IsError: false,
		},
	}

	state := NewAgentState()
	state.AddMessage(message.UserMessage{Role: "user"})

	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		Tools:      []AgentTool{bashTool},
		LoopConfig: AgentLoopConfig{MaxTurns: 2},
	}

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, nil))

	var turnStarts int
	for _, evt := range events {
		if _, ok := evt.(TurnStartEvent); ok {
			turnStarts++
		}
	}
	assert.Equal(t, 2, turnStarts, "should stop after maxTurns")
}

func TestExecuteToolCallsSequential(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}},
		{Type: "toolCall", ID: "c2", Name: "bash", Arguments: map[string]any{"cmd": "pwd"}},
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "output"}},
			IsError: false,
		},
	}

	opts := AgentOptions{
		Tools:      []AgentTool{tool},
		LoopConfig: AgentLoopConfig{},
	}

	out := NewAgentEventStream()
	go func() {
		defer out.Close()
	}()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, "c1", results[0].ToolCallID)
	assert.Equal(t, "c2", results[1].ToolCallID)
	assert.False(t, results[0].IsError)
	assert.False(t, results[1].IsError)
}

func TestExecuteToolCallsBeforeHookShortCircuits(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{"cmd": "rm -rf /"}},
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "should not reach"}},
			IsError: false,
		},
	}

	var beforeCalled bool
	opts := AgentOptions{
		Tools: []AgentTool{tool},
		BeforeToolCall: func(_ context.Context, _ BeforeToolCallContext) (*AgentToolResult, error) {
			beforeCalled = true
			return &AgentToolResult{
				Content: []message.ContentBlock{message.TextContent{Text: "blocked"}},
				IsError: true,
			}, nil
		},
	}

	out := NewAgentEventStream()
	var events []AgentEvent
	done := make(chan struct{})
	go func() {
		defer close(done)
		for evt := range out.Events() {
			events = append(events, evt)
		}
	}()

	results := executeToolCalls(context.Background(), opts, calls, out)
	out.Close()
	<-done

	assert.True(t, beforeCalled)
	assert.Equal(t, 1, len(results))
	assert.True(t, results[0].IsError)
	assert.Equal(t, "blocked", results[0].Content[0].(message.TextContent).Text)

	var endEvt ToolExecEndEvent
	for _, evt := range events {
		if e, ok := evt.(ToolExecEndEvent); ok {
			endEvt = e
		}
	}
	assert.True(t, endEvt.IsError)
}

func TestExecuteToolCallsBeforeHookReturnsNilContinues(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}},
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "file.txt"}},
			IsError: false,
		},
	}

	var beforeCalled bool
	opts := AgentOptions{
		Tools: []AgentTool{tool},
		BeforeToolCall: func(_ context.Context, _ BeforeToolCallContext) (*AgentToolResult, error) {
			beforeCalled = true
			return nil, nil
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.True(t, beforeCalled)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "file.txt", results[0].Content[0].(message.TextContent).Text)
	assert.False(t, results[0].IsError)
}

func TestExecuteToolCallsBeforeHookError(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{}},
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "should not reach"}},
		},
	}

	var beforeCalled bool
	opts := AgentOptions{
		Tools: []AgentTool{tool},
		BeforeToolCall: func(_ context.Context, _ BeforeToolCallContext) (*AgentToolResult, error) {
			beforeCalled = true
			return nil, errors.New("policy denied")
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.True(t, beforeCalled)
	assert.Equal(t, 1, len(results))
	assert.True(t, results[0].IsError)
	assert.Equal(t, "policy denied", results[0].Content[0].(message.TextContent).Text)
}

func TestExecuteToolCallsAfterHook(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}},
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "file.txt"}},
			IsError: false,
		},
	}

	var afterCtx AfterToolCallContext
	opts := AgentOptions{
		Tools: []AgentTool{tool},
		AfterToolCall: func(_ context.Context, atx AfterToolCallContext) {
			afterCtx = atx
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.Equal(t, 1, len(results))
	assert.Equal(t, "bash", afterCtx.ToolName)
	assert.Equal(t, "c1", afterCtx.ToolCall.ID)
	assert.Equal(t, "file.txt", afterCtx.Result.Content[0].(message.TextContent).Text)
	assert.False(t, afterCtx.Result.IsError)
}

func TestExecuteToolCallsAfterHookOnError(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "missing", Arguments: map[string]any{}},
	}

	var afterCalled bool
	var afterCtx AfterToolCallContext
	opts := AgentOptions{
		Tools: []AgentTool{},
		AfterToolCall: func(_ context.Context, atx AfterToolCallContext) {
			afterCalled = true
			afterCtx = atx
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.True(t, afterCalled)
	assert.Equal(t, 1, len(results))
	assert.True(t, afterCtx.Result.IsError)
	assert.Equal(t, "tool not found: missing", afterCtx.Result.Content[0].(message.TextContent).Text)
}

func TestExecuteToolCallsParallel(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}},
		{Type: "toolCall", ID: "c2", Name: "bash", Arguments: map[string]any{"cmd": "pwd"}},
		{Type: "toolCall", ID: "c3", Name: "bash", Arguments: map[string]any{"cmd": "whoami"}},
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "output"}},
			IsError: false,
		},
	}

	opts := AgentOptions{
		Tools: []AgentTool{tool},
		LoopConfig: AgentLoopConfig{
			ParallelToolCalls: true,
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.Equal(t, 3, len(results))
	assert.Equal(t, "c1", results[0].ToolCallID)
	assert.Equal(t, "c2", results[1].ToolCallID)
	assert.Equal(t, "c3", results[2].ToolCallID)
	for _, r := range results {
		assert.False(t, r.IsError)
	}
}

func TestExecuteToolCallsParallelResultOrdering(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "slow", Arguments: map[string]any{}},
		{Type: "toolCall", ID: "c2", Name: "fast", Arguments: map[string]any{}},
	}

	slowTool := &MockTool{
		ToolInfo: provider.Tool{Name: "slow"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "slow-result"}},
			IsError: false,
		},
	}
	fastTool := &MockTool{
		ToolInfo: provider.Tool{Name: "fast"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "fast-result"}},
			IsError: false,
		},
	}

	opts := AgentOptions{
		Tools: []AgentTool{slowTool, fastTool},
		LoopConfig: AgentLoopConfig{
			ParallelToolCalls: true,
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, "c1", results[0].ToolCallID)
	assert.Equal(t, "slow-result", results[0].Content[0].(message.TextContent).Text)
	assert.Equal(t, "c2", results[1].ToolCallID)
	assert.Equal(t, "fast-result", results[1].Content[0].(message.TextContent).Text)
}

func TestExecuteToolCallsParallelWithErrors(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "ok", Arguments: map[string]any{}},
		{Type: "toolCall", ID: "c2", Name: "missing", Arguments: map[string]any{}},
		{Type: "toolCall", ID: "c3", Name: "fail", Arguments: map[string]any{}},
	}

	okTool := &MockTool{
		ToolInfo: provider.Tool{Name: "ok"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "success"}},
			IsError: false,
		},
	}
	failTool := &MockTool{
		ToolInfo: provider.Tool{Name: "fail"},
		Err:      errors.New("tool blew up"),
	}

	opts := AgentOptions{
		Tools: []AgentTool{okTool, failTool},
		LoopConfig: AgentLoopConfig{
			ParallelToolCalls: true,
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.Equal(t, 3, len(results))
	assert.Equal(t, "c1", results[0].ToolCallID)
	assert.False(t, results[0].IsError)
	assert.Equal(t, "c2", results[1].ToolCallID)
	assert.True(t, results[1].IsError)
	assert.Equal(t, "c3", results[2].ToolCallID)
	assert.True(t, results[2].IsError)
}

func TestExecuteToolCallsParallelSingleCallFallsBackToSequential(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{}},
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "ok"}},
			IsError: false,
		},
	}

	opts := AgentOptions{
		Tools: []AgentTool{tool},
		LoopConfig: AgentLoopConfig{
			ParallelToolCalls: true,
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.Equal(t, 1, len(results))
	assert.Equal(t, "c1", results[0].ToolCallID)
}

func TestExecuteToolCallsParallelAfterHooks(t *testing.T) {
	calls := []message.ToolCall{
		{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{}},
		{Type: "toolCall", ID: "c2", Name: "read", Arguments: map[string]any{}},
	}

	bashTool := &MockTool{
		ToolInfo: provider.Tool{Name: "bash"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "out1"}},
			IsError: false,
		},
	}
	readTool := &MockTool{
		ToolInfo: provider.Tool{Name: "read"},
		Result: &AgentToolResult{
			Content: []message.ContentBlock{message.TextContent{Text: "out2"}},
			IsError: false,
		},
	}

	var mu sync.Mutex
	var afterCalls []string
	opts := AgentOptions{
		Tools: []AgentTool{bashTool, readTool},
		AfterToolCall: func(_ context.Context, atx AfterToolCallContext) {
			mu.Lock()
			defer mu.Unlock()
			afterCalls = append(afterCalls, atx.ToolCall.ID)
		},
		LoopConfig: AgentLoopConfig{
			ParallelToolCalls: true,
		},
	}

	out := NewAgentEventStream()
	go func() { defer out.Close() }()

	results := executeToolCalls(context.Background(), opts, calls, out)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, 2, len(afterCalls))
	assert.Contains(t, afterCalls, "c1")
	assert.Contains(t, afterCalls, "c2")
}

func mockStreamFn(msg message.AssistantMessage) provider.StreamFn {
	return func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		return mockTextStream(msg, []string{extractText(msg)})
	}
}

func TestNewAgent(t *testing.T) {
	a := NewAgent(AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}, mockStreamFn(message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "hi"}},
		StopReason: message.StopReasonStop,
	}))

	assert.NotNil(t, a)
	assert.NotNil(t, a.State())
	assert.False(t, a.State().IsStreaming())
}

func TestAgentPrompt(t *testing.T) {
	response := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "hello there"}},
		StopReason: message.StopReasonStop,
	}
	a := NewAgent(AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}, mockStreamFn(response))

	events := collectEvents(a.Prompt(context.Background(), "hi"))

	var hasAgentStart, hasAgentEnd bool
	for _, evt := range events {
		switch evt.(type) {
		case AgentStartEvent:
			hasAgentStart = true
		case AgentEndEvent:
			hasAgentEnd = true
		}
	}
	assert.True(t, hasAgentStart)
	assert.True(t, hasAgentEnd)
	assert.Equal(t, 2, len(a.State().GetMessages()))
}

func TestAgentSubscribe(t *testing.T) {
	response := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "hi"}},
		StopReason: message.StopReasonStop,
	}
	a := NewAgent(AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}, mockStreamFn(response))

	sub := a.Subscribe()

	collectEvents(a.Prompt(context.Background(), "hi"))

	var subEvents []AgentEvent
	for evt := range sub {
		subEvents = append(subEvents, evt)
	}
	assert.True(t, len(subEvents) > 0, "subscriber should receive events")
}

func TestAgentAbort(t *testing.T) {
	response := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "hi"}},
		StopReason: message.StopReasonStop,
	}
	a := NewAgent(AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}, mockStreamFn(response))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := a.Prompt(ctx, "hi")

	a.Abort()

	var gotAny bool
	for evt := range ch {
		gotAny = true
		_ = evt
		break
	}
	_ = gotAny
}

func TestAgentStateReturnsState(t *testing.T) {
	a := NewAgent(AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}, mockStreamFn(message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "ok"}},
		StopReason: message.StopReasonStop,
	}))

	s := a.State()
	assert.Equal(t, 0, len(s.GetMessages()))
	assert.False(t, s.IsStreaming())
}

func TestAgentFollowUp(t *testing.T) {
	var callCount int64
	responses := []message.AssistantMessage{
		{
			Role:       "assistant",
			Content:    []message.ContentBlock{message.TextContent{Text: "first"}},
			StopReason: message.StopReasonStop,
		},
		{
			Role:       "assistant",
			Content:    []message.ContentBlock{message.TextContent{Text: "second"}},
			StopReason: message.StopReasonStop,
		},
	}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		idx := int(atomic.AddInt64(&callCount, 1) - 1)
		return mockTextStream(responses[idx], []string{extractText(responses[idx])})
	}

	a := NewAgent(AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}, streamFn)

	a.FollowUp(context.Background(), "follow up")
	ch := a.Prompt(context.Background(), "hello")

	collectEvents(ch)

	assert.Equal(t, int64(2), atomic.LoadInt64(&callCount))
}

func TestAgentSteer(t *testing.T) {
	a := NewAgent(AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}, mockStreamFn(message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "ok"}},
		StopReason: message.StopReasonStop,
	}))

	a.Steer(context.Background(), "new direction")
	assert.Equal(t, 1, a.queue.Len())
}

func TestStreamProxyRoutesToRegisteredProvider(t *testing.T) {
	response := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "proxied"}},
		StopReason: message.StopReasonStop,
	}

	provider.ResetRegistry()
	defer provider.ResetRegistry()

	provider.Register(provider.Provider{
		API: "test-api",
		Stream: func(m provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
			return mockTextStream(response, []string{"proxied"})
		},
	})

	stream := StreamProxy(
		provider.Model{ID: "test-model", API: "test-api"},
		provider.Context{},
		nil,
	)

	var gotText string
	for evt := range stream.Events() {
		if td, ok := evt.(provider.EventTextDelta); ok {
			gotText += td.Delta
		}
	}
	assert.Equal(t, "proxied", gotText)
}

func TestStreamProxyReturnsEmptyStreamForUnknownProvider(t *testing.T) {
	provider.ResetRegistry()
	defer provider.ResetRegistry()

	stream := StreamProxy(
		provider.Model{ID: "unknown-model", API: "nonexistent"},
		provider.Context{},
		nil,
	)

	evt, ok := <-stream.Events()
	assert.True(t, ok, "expected EventError on stream for unknown provider")
	_, isError := evt.(provider.EventError)
	assert.True(t, isError, "expected EventError event for unknown provider")

	_, ok = <-stream.Events()
	assert.False(t, ok, "expected stream to close after error event")
}

func TestStreamProxyPassesArgumentsToProvider(t *testing.T) {
	response := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "ok"}},
		StopReason: message.StopReasonStop,
	}

	provider.ResetRegistry()
	defer provider.ResetRegistry()

	var gotModel provider.Model
	var gotCtx provider.Context
	var gotOpts *provider.StreamOptions

	provider.Register(provider.Provider{
		API: "capture-api",
		Stream: func(m provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
			gotModel = m
			gotCtx = ctx
			gotOpts = opts
			return mockTextStream(response, []string{"ok"})
		},
	})

	model := provider.Model{ID: "my-model", API: "capture-api"}
	ctx := provider.Context{
		SystemPrompt: "you are helpful",
	}
	temp := 0.7
	opts := &provider.StreamOptions{Temperature: &temp}

	StreamProxy(model, ctx, opts)

	assert.Equal(t, "my-model", gotModel.ID)
	assert.Equal(t, "capture-api", gotModel.API)
	assert.Equal(t, "you are helpful", gotCtx.SystemPrompt)
	assert.NotNil(t, gotOpts)
	assert.Equal(t, 0.7, *gotOpts.Temperature)
}

func TestAgentLoopSteeringInterruptsBetweenTurns(t *testing.T) {
	var callCount int64
	toolCallMsg := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.TextContent{Text: "using tool"},
			message.ToolCall{
				ID:   "tc1",
				Name: "test_tool",
				Arguments: map[string]any{
					"query": "hello",
				},
			},
		},
		StopReason: message.StopReasonToolUse,
	}
	steeredResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "steered"}},
		StopReason: message.StopReasonStop,
	}

	responses := []message.AssistantMessage{toolCallMsg, steeredResponse}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		idx := int(atomic.AddInt64(&callCount, 1) - 1)
		return mockTextStream(responses[idx], []string{extractText(responses[idx])})
	}

	tool := &MockTool{
		ToolInfo: provider.Tool{Name: "test_tool", Description: "test"},
		Result:   &AgentToolResult{Content: []message.ContentBlock{message.TextContent{Text: "result"}}},
	}

	state := NewAgentState()
	queue := NewPendingMessageQueue()
	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		Tools:      []AgentTool{tool},
		LoopConfig: AgentLoopConfig{MaxTurns: 5},
	}

	steerMsg := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{message.TextContent{Text: "change direction"}},
	}
	queue.Enqueue(QueueModeSteering, steerMsg)

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, queue))

	var turnEnds []TurnEndEvent
	for _, evt := range events {
		if te, ok := evt.(TurnEndEvent); ok {
			turnEnds = append(turnEnds, te)
		}
	}

	assert.Equal(t, int64(2), atomic.LoadInt64(&callCount))

	msgs := state.GetMessages()
	hasSteerMsg := false
	for _, m := range msgs {
		if um, ok := m.(message.UserMessage); ok {
			if len(um.Content) > 0 {
				if tc, ok := um.Content[0].(message.TextContent); ok && tc.Text == "change direction" {
					hasSteerMsg = true
				}
			}
		}
	}
	assert.True(t, hasSteerMsg, "steering message should be injected into state")
	assert.True(t, len(turnEnds) >= 2, "should have multiple turns after steering")
}

func TestAgentLoopFollowUpAfterCompletion(t *testing.T) {
	var callCount int64
	firstResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "first"}},
		StopReason: message.StopReasonStop,
	}
	secondResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "second"}},
		StopReason: message.StopReasonStop,
	}

	responses := []message.AssistantMessage{firstResponse, secondResponse}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		idx := int(atomic.AddInt64(&callCount, 1) - 1)
		return mockTextStream(responses[idx], []string{extractText(responses[idx])})
	}

	state := NewAgentState()
	queue := NewPendingMessageQueue()
	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}

	followUpMsg := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{message.TextContent{Text: "follow up"}},
	}
	queue.Enqueue(QueueModeFollowUp, followUpMsg)

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn, queue))

	var endEvents []AgentEndEvent
	for _, evt := range events {
		if ae, ok := evt.(AgentEndEvent); ok {
			endEvents = append(endEvents, ae)
		}
	}

	assert.Equal(t, int64(2), atomic.LoadInt64(&callCount))
	assert.Equal(t, 1, len(endEvents))

	msgs := state.GetMessages()
	hasFollowUp := false
	for _, m := range msgs {
		if um, ok := m.(message.UserMessage); ok {
			if len(um.Content) > 0 {
				if tc, ok := um.Content[0].(message.TextContent); ok && tc.Text == "follow up" {
					hasFollowUp = true
				}
			}
		}
	}
	assert.True(t, hasFollowUp, "follow-up message should be in state")
}

func TestAgentLoopSteeringTakesPriorityOverFollowUp(t *testing.T) {
	var callCount int64
	firstResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "first"}},
		StopReason: message.StopReasonStop,
	}
	secondResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "second"}},
		StopReason: message.StopReasonStop,
	}
	thirdResponse := message.AssistantMessage{
		Role:       "assistant",
		Content:    []message.ContentBlock{message.TextContent{Text: "third"}},
		StopReason: message.StopReasonStop,
	}

	responses := []message.AssistantMessage{firstResponse, secondResponse, thirdResponse}

	streamFn := func(_ provider.Model, _ provider.Context, _ *provider.StreamOptions) *provider.AssistantMessageEventStream {
		idx := int(atomic.AddInt64(&callCount, 1) - 1)
		return mockTextStream(responses[idx], []string{extractText(responses[idx])})
	}

	state := NewAgentState()
	queue := NewPendingMessageQueue()
	opts := AgentOptions{
		Model:      provider.Model{ID: "test", API: "test"},
		LoopConfig: AgentLoopConfig{MaxTurns: 1},
	}

	followUpMsg := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{message.TextContent{Text: "follow up"}},
	}
	steerMsg := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{message.TextContent{Text: "steer"}},
	}
	queue.Enqueue(QueueModeFollowUp, followUpMsg)
	queue.Enqueue(QueueModeSteering, steerMsg)

	collectEvents(AgentLoop(context.Background(), opts, state, streamFn, queue))

	assert.Equal(t, int64(3), atomic.LoadInt64(&callCount))

	msgs := state.GetMessages()
	var userTexts []string
	for _, m := range msgs {
		if um, ok := m.(message.UserMessage); ok {
			if len(um.Content) > 0 {
				if tc, ok := um.Content[0].(message.TextContent); ok {
					userTexts = append(userTexts, tc.Text)
				}
			}
		}
	}
	assert.Equal(t, []string{"steer", "follow up"}, userTexts, "steering should be processed before follow-up")
}
