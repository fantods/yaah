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

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

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

	events := collectEvents(AgentLoop(context.Background(), opts, state, streamFn))

	var turnStarts int
	for _, evt := range events {
		if _, ok := evt.(TurnStartEvent); ok {
			turnStarts++
		}
	}
	assert.Equal(t, 2, turnStarts, "should stop after maxTurns")
}
