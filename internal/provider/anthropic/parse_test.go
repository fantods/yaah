package anthropic

import (
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPartial() *message.AssistantMessage {
	return &message.AssistantMessage{Role: "assistant", Content: []message.ContentBlock{}}
}

func TestHandleEventMessageStart(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type: "message_start",
		Message: anthropic.Message{
			ID:    "msg_123",
			Model: "claude-3",
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	count := 0
	for e := range events {
		if _, ok := e.(provider.EventStart); ok {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestHandleEventContentBlockStartText(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type:  "content_block_start",
		Index: 0,
		ContentBlock: anthropic.ContentBlockStartEventContentBlockUnion{
			Type: "text",
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	ts, ok := e.(provider.EventTextStart)
	require.True(t, ok)
	assert.Equal(t, 0, ts.ContentIndex)
}

func TestHandleEventContentBlockStartToolUse(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type:  "content_block_start",
		Index: 2,
		ContentBlock: anthropic.ContentBlockStartEventContentBlockUnion{
			Type: "tool_use",
			ID:   "toolu_abc",
			Name: "read_file",
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	tcs, ok := e.(provider.EventToolCallStart)
	require.True(t, ok)
	assert.Equal(t, 2, tcs.ContentIndex)
}

func TestHandleEventContentBlockStartThinking(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type:  "content_block_start",
		Index: 1,
		ContentBlock: anthropic.ContentBlockStartEventContentBlockUnion{
			Type: "thinking",
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	ts, ok := e.(provider.EventThinkingStart)
	require.True(t, ok)
	assert.Equal(t, 1, ts.ContentIndex)
}

func TestHandleEventContentBlockDeltaText(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type:  "content_block_delta",
		Index: 0,
		Delta: anthropic.MessageStreamEventUnionDelta{
			Type: "text_delta",
			Text: "hello ",
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	td, ok := e.(provider.EventTextDelta)
	require.True(t, ok)
	assert.Equal(t, 0, td.ContentIndex)
	assert.Equal(t, "hello ", td.Delta)
}

func TestHandleEventContentBlockDeltaThinking(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type:  "content_block_delta",
		Index: 1,
		Delta: anthropic.MessageStreamEventUnionDelta{
			Type:     "thinking_delta",
			Thinking: "let me think",
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	td, ok := e.(provider.EventThinkingDelta)
	require.True(t, ok)
	assert.Equal(t, 1, td.ContentIndex)
	assert.Equal(t, "let me think", td.Delta)
}

func TestHandleEventContentBlockDeltaInputJSON(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type:  "content_block_delta",
		Index: 2,
		Delta: anthropic.MessageStreamEventUnionDelta{
			Type:        "input_json_delta",
			PartialJSON: `{"path":`,
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	td, ok := e.(provider.EventToolCallDelta)
	require.True(t, ok)
	assert.Equal(t, 2, td.ContentIndex)
	assert.Equal(t, `{"path":`, td.Delta)
}

func TestHandleEventContentBlockStopText(t *testing.T) {
	partial := newPartial()
	partial.Content = append(partial.Content, message.TextContent{Type: "text", Text: "hello"})
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type:  "content_block_stop",
		Index: 0,
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	te, ok := e.(provider.EventTextEnd)
	require.True(t, ok)
	assert.Equal(t, 0, te.ContentIndex)
}

func TestHandleEventMessageDelta(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type: "message_delta",
		Delta: anthropic.MessageStreamEventUnionDelta{
			StopReason: "end_turn",
		},
		Usage: anthropic.MessageDeltaUsage{
			OutputTokens: 50,
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	done, ok := e.(provider.EventDone)
	require.True(t, ok)
	assert.Equal(t, message.StopReasonStop, done.Reason)
}

func TestHandleEventMessageDeltaToolUse(t *testing.T) {
	partial := newPartial()
	events := make(chan provider.AssistantMessageEvent, 10)

	evt := anthropic.MessageStreamEventUnion{
		Type: "message_delta",
		Delta: anthropic.MessageStreamEventUnionDelta{
			StopReason: "tool_use",
		},
	}

	HandleEvent(evt, partial, events)
	close(events)

	e := <-events
	done, ok := e.(provider.EventDone)
	require.True(t, ok)
	assert.Equal(t, message.StopReasonToolUse, done.Reason)
}

func TestMapStopReason(t *testing.T) {
	assert.Equal(t, message.StopReasonStop, mapStopReason("end_turn"))
	assert.Equal(t, message.StopReasonLength, mapStopReason("max_tokens"))
	assert.Equal(t, message.StopReasonToolUse, mapStopReason("tool_use"))
	assert.Equal(t, message.StopReasonStop, mapStopReason("stop_sequence"))
	assert.Equal(t, message.StopReasonStop, mapStopReason("unknown"))
}
