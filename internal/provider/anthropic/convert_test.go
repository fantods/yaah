package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, "user", string(params[0].Role))

	blocks := params[0].Content
	require.Len(t, blocks, 1)
	text := blocks[0].OfText
	require.NotNil(t, text)
	assert.Equal(t, "hello", text.Text)
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

	blocks := params[0].Content
	require.Len(t, blocks, 1)
	img := blocks[0].OfImage
	require.NotNil(t, img)
	require.NotNil(t, img.Source)
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
	assert.Equal(t, "assistant", string(params[0].Role))

	blocks := params[0].Content
	require.Len(t, blocks, 1)
	text := blocks[0].OfText
	require.NotNil(t, text)
	assert.Equal(t, "hi there", text.Text)
}

func TestConvertAssistantMessageThinking(t *testing.T) {
	msgs := []message.Message{
		message.AssistantMessage{
			Role: "assistant",
			Content: []message.ContentBlock{
				message.ThinkingContent{Type: "thinking", Thinking: "hmm", ThinkingSignature: "sig123"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)

	blocks := params[0].Content
	require.Len(t, blocks, 1)
	thinking := blocks[0].OfThinking
	require.NotNil(t, thinking)
	assert.Equal(t, "hmm", thinking.Thinking)
	assert.Equal(t, "sig123", thinking.Signature)
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

	blocks := params[0].Content
	require.Len(t, blocks, 2)

	text := blocks[0].OfText
	require.NotNil(t, text)
	assert.Equal(t, "let me check", text.Text)

	toolUse := blocks[1].OfToolUse
	require.NotNil(t, toolUse)
	assert.Equal(t, "call_abc", toolUse.ID)
	assert.Equal(t, "read_file", toolUse.Name)
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

	trBlocks := params[1].Content
	require.Len(t, trBlocks, 1)
	result := trBlocks[0].OfToolResult
	require.NotNil(t, result)
	assert.Equal(t, "call_1", result.ToolUseID)
}

func TestConvertToolResultMessageIsError(t *testing.T) {
	msgs := []message.Message{
		message.ToolResultMessage{
			Role:       "toolResult",
			ToolCallID: "call_err",
			IsError:    true,
			Content: []message.ContentBlock{
				message.TextContent{Type: "text", Text: "command failed"},
			},
		},
	}

	params := ConvertMessages(msgs)
	require.Len(t, params, 1)

	result := params[0].Content[0].OfToolResult
	require.NotNil(t, result)
	assert.True(t, result.IsError.Valid() && result.IsError.Value == true)
}

func TestConvertTools(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)
	tools := []provider.Tool{
		{Name: "read_file", Description: "Read a file", Parameters: schema},
	}

	params := ConvertTools(tools)
	require.Len(t, params, 1)
	assert.Equal(t, "read_file", params[0].Name)
}

func TestConvertEmptyMessages(t *testing.T) {
	params := ConvertMessages([]message.Message{})
	assert.Empty(t, params)
}

func TestConvertEmptyTools(t *testing.T) {
	params := ConvertTools([]provider.Tool{})
	assert.Empty(t, params)
}
