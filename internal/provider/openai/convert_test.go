package openai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func TestConvertUserMessageText(t *testing.T) {
	msgs := []message.Message{
		message.UserMessage{
			Role: "user",
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "hello"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)

	// Verify it's a user message by checking OfUser is set
	assert.NotNil(t, params[0].OfUser)
}

func TestConvertUserMessageImage(t *testing.T) {
	msgs := []message.Message{
		message.UserMessage{
			Role: "user",
			Content: []message.ContentBlock{
				message.ImageContent{Type: "image", Data: "aGVsbG8=", MIMEType: "image/png"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfUser)
}

func TestConvertUserMessageMixed(t *testing.T) {
	msgs := []message.Message{
		message.UserMessage{
			Role: "user",
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "look at this"},
				message.ImageContent{Type: "image", Data: "aGVsbG8=", MIMEType: "image/png"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfUser)
}

func TestConvertAssistantMessageText(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Role: "assistant",
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "hi there"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfAssistant)
}

func TestConvertAssistantMessageToolCall(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Role: "assistant",
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "let me check"},
				message.ToolCall{
					Type:      "toolCall",
					ID:        "call_abc",
					Name:      "read_file",
					Arguments: map[string]any{"path": "/tmp/x"},
				},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfAssistant)
	assert.Len(t, params[0].OfAssistant.ToolCalls, 1)
}

func TestConvertToolResultMessage(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Role: "assistant",
			Content: []message.ContentBlock{
				message.ToolCall{Type: "toolCall", ID: "call_1", Name: "bash", Arguments: map[string]any{}},
			},
		},
		message.ToolResultMessage{
			Role:       "toolResult",
			ToolCallID: "call_1",
			ToolName:   "bash",
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "output here"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 2)
	assert.NotNil(t, params[1].OfTool)
	assert.Equal(t, "call_1", params[1].OfTool.ToolCallID)
}

func TestConvertTools(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)
	tools := []provider.Tool{
		{Name: "read_file", Description: "Read a file", Parameters: schema},
	}

	params := ConvertTools(tools)
	require.Len(t, params, 1)
	assert.Equal(t, "read_file", params[0].Function.Name)
	assert.Equal(t, "Read a file", params[0].Function.Description.Value)
}

func TestConvertEmptyMessages(t *testing.T) {
	params := ConvertMessages([]message.Message{})
	assert.Empty(t, params)
}

func TestConvertEmptyTools(t *testing.T) {
	params := ConvertTools([]provider.Tool{})
	assert.Empty(t, params)
}

func TestConvertAssistantMessageThinking(t *testing.T) {
	// Thinking blocks should be silently ignored
	msgs := []message.Message{
		message.AssistantMessage{
			Role: "assistant",
			Content: []message.ContentBlock{
				message.ThinkingContent{Type: "thinking", Thinking: "internal thoughts"},
				message.TextContent{Type: "text", Text: "final answer"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfAssistant)
	// Content should only have the text, not thinking
	assert.True(t, params[0].OfAssistant.Content.OfString.Valid())
	assert.Equal(t, "final answer", params[0].OfAssistant.Content.OfString.Value)
}

func TestConvertToolResultMessageEmpty(t *testing.T) {
	// Empty tool results should get placeholder content
	msgs := []message.Message{
		message.ToolResultMessage{
			Role:       "toolResult",
			ToolCallID: "call_1",
			ToolName:   "bash",
			Content:    []message.ContentBlock{},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfTool)
	// Should have placeholder content
	assert.True(t, params[0].OfTool.Content.OfString.Valid())
	assert.Equal(t, "(no output)", params[0].OfTool.Content.OfString.Value)
}

func TestConvertMultipleToolCalls(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Role: "assistant",
			Content: []message.ContentBlock{
				message.ToolCall{Type: "toolCall", ID: "call_1", Name: "read", Arguments: map[string]any{"path": "/a"}},
				message.ToolCall{Type: "toolCall", ID: "call_2", Name: "read", Arguments: map[string]any{"path": "/b"}},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfAssistant)
	assert.Len(t, params[0].OfAssistant.ToolCalls, 2)
	assert.Equal(t, "call_1", params[0].OfAssistant.ToolCalls[0].ID)
	assert.Equal(t, "call_2", params[0].OfAssistant.ToolCalls[1].ID)
}

func TestConvertAssistantMessageEmptyText(t *testing.T) {
	// Empty/whitespace text should be filtered
	msgs := []message.Message{
		message.AssistantMessage{
			Role: "assistant",
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "   "},
				message.TextContent{Type: "text", Text: ""},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfAssistant)
	// Content should be empty string (not whitespace)
	assert.Equal(t, "", params[0].OfAssistant.Content.OfString.Value)
}

func TestConvertUserMessageEmptyContent(t *testing.T) {
	msgs := []message.Message{
		message.UserMessage{
			Role:    "user",
			Content: []message.ContentBlock{},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)
	assert.NotNil(t, params[0].OfUser)
}
