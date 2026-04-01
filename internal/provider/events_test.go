package provider

import (
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/stretchr/testify/assert"
)

func TestEventStartImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventStart{}
}

func TestEventTextStartImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventTextStart{}
}

func TestEventTextDeltaImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventTextDelta{}
}

func TestEventTextEndImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventTextEnd{}
}

func TestEventThinkingStartImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventThinkingStart{}
}

func TestEventThinkingDeltaImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventThinkingDelta{}
}

func TestEventThinkingEndImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventThinkingEnd{}
}

func TestEventToolCallStartImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventToolCallStart{}
}

func TestEventToolCallDeltaImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventToolCallDelta{}
}

func TestEventToolCallEndImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventToolCallEnd{}
}

func TestEventDoneImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventDone{}
}

func TestEventErrorImplementsInterface(t *testing.T) {
	var _ AssistantMessageEvent = EventError{}
}

func TestEventStartFieldAccess(t *testing.T) {
	partial := &message.AssistantMessage{Role: "assistant"}
	evt := EventStart{Partial: partial}

	assert.Equal(t, "assistant", evt.Partial.Role)
}

func TestEventTextDeltaFieldAccess(t *testing.T) {
	partial := &message.AssistantMessage{}
	evt := EventTextDelta{
		Partial:      partial,
		ContentIndex: 2,
		Delta:        "hello",
	}

	assert.Equal(t, partial, evt.Partial)
	assert.Equal(t, 2, evt.ContentIndex)
	assert.Equal(t, "hello", evt.Delta)
}

func TestEventThinkingEndFieldAccess(t *testing.T) {
	partial := &message.AssistantMessage{}
	evt := EventThinkingEnd{
		Partial:      partial,
		ContentIndex: 1,
		Content:      "I considered...",
	}

	assert.Equal(t, 1, evt.ContentIndex)
	assert.Equal(t, "I considered...", evt.Content)
}

func TestEventToolCallEndFieldAccess(t *testing.T) {
	partial := &message.AssistantMessage{}
	tc := message.ToolCall{
		Type:      "toolCall",
		ID:        "call_abc",
		Name:      "read_file",
		Arguments: map[string]any{"path": "/tmp/x"},
	}
	evt := EventToolCallEnd{
		Partial:      partial,
		ContentIndex: 0,
		ToolCall:     tc,
	}

	assert.Equal(t, 0, evt.ContentIndex)
	assert.Equal(t, "call_abc", evt.ToolCall.ID)
	assert.Equal(t, "read_file", evt.ToolCall.Name)
	assert.Equal(t, "/tmp/x", evt.ToolCall.Arguments["path"])
}

func TestEventDoneFieldAccess(t *testing.T) {
	msg := message.AssistantMessage{
		Role:       "assistant",
		StopReason: message.StopReasonStop,
		Usage:      message.Usage{Input: 100, Output: 50},
	}
	evt := EventDone{Reason: message.StopReasonStop, Message: msg}

	assert.Equal(t, message.StopReasonStop, evt.Reason)
	assert.Equal(t, int64(100), evt.Message.Usage.Input)
	assert.Equal(t, int64(50), evt.Message.Usage.Output)
}

func TestEventErrorFieldAccess(t *testing.T) {
	msg := message.AssistantMessage{
		Role:         "assistant",
		StopReason:   message.StopReasonError,
		ErrorMessage: "rate limited",
	}
	evt := EventError{Reason: message.StopReasonError, Message: msg}

	assert.Equal(t, message.StopReasonError, evt.Reason)
	assert.Equal(t, "rate limited", evt.Message.ErrorMessage)
}

func TestPartialPointerMutation(t *testing.T) {
	partial := &message.AssistantMessage{Model: "claude-3"}
	evt := EventTextDelta{Partial: partial, Delta: "hi"}

	evt.Partial.Model = "gpt-4"
	assert.Equal(t, "gpt-4", partial.Model)

	partial.Model = "claude-4"
	assert.Equal(t, "claude-4", evt.Partial.Model)
}

func TestTypeSwitchExhaustiveness(t *testing.T) {
	partial := &message.AssistantMessage{}
	tc := message.ToolCall{Type: "toolCall", ID: "c1", Name: "bash", Arguments: map[string]any{}}

	events := []AssistantMessageEvent{
		EventStart{Partial: partial},
		EventTextStart{Partial: partial, ContentIndex: 0},
		EventTextDelta{Partial: partial, ContentIndex: 0, Delta: "x"},
		EventTextEnd{Partial: partial, ContentIndex: 0, Content: "x"},
		EventThinkingStart{Partial: partial, ContentIndex: 1},
		EventThinkingDelta{Partial: partial, ContentIndex: 1, Delta: "hmm"},
		EventThinkingEnd{Partial: partial, ContentIndex: 1, Content: "hmm"},
		EventToolCallStart{Partial: partial, ContentIndex: 2},
		EventToolCallDelta{Partial: partial, ContentIndex: 2, Delta: `{"a":`},
		EventToolCallEnd{Partial: partial, ContentIndex: 2, ToolCall: tc},
		EventDone{Reason: message.StopReasonStop, Message: message.AssistantMessage{}},
		EventError{Reason: message.StopReasonError, Message: message.AssistantMessage{}},
	}

	for i, evt := range events {
		switch evt.(type) {
		case EventStart:
		case EventTextStart:
		case EventTextDelta:
		case EventTextEnd:
		case EventThinkingStart:
		case EventThinkingDelta:
		case EventThinkingEnd:
		case EventToolCallStart:
		case EventToolCallDelta:
		case EventToolCallEnd:
		case EventDone:
		case EventError:
		default:
			t.Fatalf("event %d (%T) did not match any case", i, evt)
		}
	}
}
