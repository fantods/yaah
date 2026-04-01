package provider

import (
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeToolCallIDEmpty(t *testing.T) {
	assert.Equal(t, "", NormalizeToolCallID(""))
}

func TestNormalizeToolCallIDAlreadyHasPrefix(t *testing.T) {
	assert.Equal(t, "call_abc123", NormalizeToolCallID("call_abc123"))
}

func TestNormalizeToolCallIDNoPrefix(t *testing.T) {
	assert.Equal(t, "call_abc123", NormalizeToolCallID("abc123"))
}

func TestNormalizeToolCallIDTooluPrefix(t *testing.T) {
	assert.Equal(t, "toolu_abc123", NormalizeToolCallID("toolu_abc123"))
}

func TestExtractThinkingBlocksEmpty(t *testing.T) {
	msg := message.AssistantMessage{
		Content: []message.ContentBlock{},
	}
	blocks, remaining := ExtractThinkingBlocks(msg)
	assert.Empty(t, blocks)
	assert.Empty(t, remaining)
}

func TestExtractThinkingBlocksNone(t *testing.T) {
	msg := message.AssistantMessage{
		Content: []message.ContentBlock{
			message.TextContent{Type: "text", Text: "hello"},
		},
	}
	blocks, remaining := ExtractThinkingBlocks(msg)
	assert.Empty(t, blocks)
	assert.Len(t, remaining, 1)
}

func TestExtractThinkingBlocksPresent(t *testing.T) {
	msg := message.AssistantMessage{
		Content: []message.ContentBlock{
			message.ThinkingContent{Type: "thinking", Thinking: "hmm"},
			message.TextContent{Type: "text", Text: "answer"},
			message.ThinkingContent{Type: "thinking", Thinking: "more"},
		},
	}
	blocks, remaining := ExtractThinkingBlocks(msg)
	assert.Len(t, blocks, 2)
	assert.Len(t, remaining, 1)

	assert.Equal(t, "hmm", blocks[0].Thinking)
	assert.Equal(t, "more", blocks[1].Thinking)

	text, ok := remaining[0].(message.TextContent)
	require.True(t, ok)
	assert.Equal(t, "answer", text.Text)
}

func TestExtractThinkingBlocksOnlyThinking(t *testing.T) {
	msg := message.AssistantMessage{
		Content: []message.ContentBlock{
			message.ThinkingContent{Type: "thinking", Thinking: "deep thought"},
		},
	}
	blocks, remaining := ExtractThinkingBlocks(msg)
	assert.Len(t, blocks, 1)
	assert.Empty(t, remaining)
}

func TestMergeThinkingBlocksEmpty(t *testing.T) {
	result := MergeThinkingBlocks(nil, []message.ContentBlock{
		message.TextContent{Type: "text", Text: "hi"},
	})
	assert.Len(t, result, 1)
}

func TestMergeThinkingBlocksBothPresent(t *testing.T) {
	thinking := []message.ThinkingContent{
		{Type: "thinking", Thinking: "hmm"},
	}
	other := []message.ContentBlock{
		message.TextContent{Type: "text", Text: "answer"},
	}
	result := MergeThinkingBlocks(thinking, other)
	assert.Len(t, result, 2)

	assert.Equal(t, "thinking", result[0].(message.ThinkingContent).Type)
	assert.Equal(t, "text", result[1].(message.TextContent).Type)
}

func TestMergeThinkingBlocksThinkingOnly(t *testing.T) {
	thinking := []message.ThinkingContent{
		{Type: "thinking", Thinking: "hmm"},
	}
	result := MergeThinkingBlocks(thinking, nil)
	assert.Len(t, result, 1)
}

func TestStripThinkingBlocksFromMessages(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Content: []message.ContentBlock{
				message.ThinkingContent{Type: "thinking", Thinking: "private"},
				message.TextContent{Type: "text", Text: "public"},
			},
		},
		message.UserMessage{
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "follow up"},
			},
		},
	}

	result := StripThinkingBlocks(msgs)
	require.Len(t, result, 2)

	asm, ok := result[0].(message.AssistantMessage)
	require.True(t, ok)
	assert.Len(t, asm.Content, 1)
	assert.Equal(t, "public", asm.Content[0].(message.TextContent).Text)

	um, ok := result[1].(message.UserMessage)
	require.True(t, ok)
	assert.Len(t, um.Content, 1)
}

func TestStripThinkingBlocksNonAssistantUnchanged(t *testing.T) {
	msgs := []message.Message{
		message.UserMessage{
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "hello"},
			},
		},
	}

	result := StripThinkingBlocks(msgs)
	assert.Equal(t, msgs, result)
}

func TestNormalizeMessagesToolCallIDs(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Content: []message.ContentBlock{
				message.ToolCall{Type: "toolCall", ID: "abc123", Name: "bash", Arguments: map[string]any{}},
			},
		},
		message.ToolResultMessage{
			ToolCallID: "abc123",
			ToolName:   "bash",
			Content:    []message.ContentBlock{message.TextContent{Type: "text", Text: "ok"}},
		},
	}

	result := NormalizeMessages(msgs)
	require.Len(t, result, 2)

	asm, ok := result[0].(message.AssistantMessage)
	require.True(t, ok)
	tc, ok := asm.Content[0].(message.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_abc123", tc.ID)

	trm, ok := result[1].(message.ToolResultMessage)
	require.True(t, ok)
	assert.Equal(t, "call_abc123", trm.ToolCallID)
}

func TestNormalizeMessagesAlreadyNormalized(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Content: []message.ContentBlock{
				message.ToolCall{Type: "toolCall", ID: "call_xyz", Name: "bash", Arguments: map[string]any{}},
			},
		},
	}

	result := NormalizeMessages(msgs)
	asm := result[0].(message.AssistantMessage)
	tc := asm.Content[0].(message.ToolCall)
	assert.Equal(t, "call_xyz", tc.ID)
}
