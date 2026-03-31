package message

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMessageImplementsMessage(t *testing.T) {
	var _ Message = UserMessage{}
}

func TestAssistantMessageImplementsMessage(t *testing.T) {
	var _ Message = AssistantMessage{}
}

func TestToolResultMessageImplementsMessage(t *testing.T) {
	var _ Message = ToolResultMessage{}
}

func TestUserMessageJSONRoundTrip(t *testing.T) {
	original := UserMessage{
		Role: "user",
		Content: []ContentBlock{
			TextContent{Type: "text", Text: "hello"},
			ImageContent{Type: "image", Data: "aGVsbG8=", MIMEType: "image/png"},
		},
		Timestamp: 1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded UserMessage
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "user", decoded.Role)
	assert.Equal(t, int64(1710000000000), decoded.Timestamp)
	assert.Len(t, decoded.Content, 2)
}

func TestAssistantMessageJSONRoundTrip(t *testing.T) {
	original := AssistantMessage{
		Role:     "assistant",
		API:      "anthropic-messages",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		Content: []ContentBlock{
			TextContent{Type: "text", Text: "hi there"},
			ThinkingContent{Type: "thinking", Thinking: "hmm"},
			ToolCall{
				Type:      "toolCall",
				ID:        "call_123",
				Name:      "bash",
				Arguments: map[string]any{"cmd": "ls"},
			},
		},
		ResponseID: "resp_abc",
		Usage: Usage{
			Input:       1000,
			Output:      500,
			TotalTokens: 1500,
			Cost:        Cost{Input: 0.003, Output: 0.0075, Total: 0.0105},
		},
		StopReason: StopReasonStop,
		Timestamp:  1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded AssistantMessage
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "assistant", decoded.Role)
	assert.Equal(t, "anthropic-messages", decoded.API)
	assert.Equal(t, "anthropic", decoded.Provider)
	assert.Equal(t, "claude-sonnet-4-20250514", decoded.Model)
	assert.Equal(t, "resp_abc", decoded.ResponseID)
	assert.Equal(t, StopReasonStop, decoded.StopReason)
	assert.Equal(t, int64(1000), decoded.Usage.Input)
	assert.Equal(t, int64(500), decoded.Usage.Output)
	assert.Equal(t, int64(1500), decoded.Usage.TotalTokens)
	assert.Equal(t, 0.0105, decoded.Usage.Cost.Total)
	assert.Equal(t, int64(1710000000000), decoded.Timestamp)
	assert.Len(t, decoded.Content, 3)
}

func TestAssistantMessageOmitEmpty(t *testing.T) {
	original := AssistantMessage{
		Role:       "assistant",
		Content:    []ContentBlock{},
		API:        "openai-completions",
		Provider:   "openai",
		Model:      "gpt-4o",
		Usage:      Usage{},
		StopReason: StopReasonStop,
		Timestamp:  1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "responseId")
	assert.NotContains(t, string(data), "errorMessage")
}

func TestAssistantMessageWithErrorMessage(t *testing.T) {
	original := AssistantMessage{
		Role:         "assistant",
		Content:      []ContentBlock{},
		API:          "openai-completions",
		Provider:     "openai",
		Model:        "gpt-4o",
		StopReason:   StopReasonError,
		ErrorMessage: "rate limit exceeded",
		Timestamp:    1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"errorMessage":"rate limit exceeded"`)
}

func TestToolResultMessageJSONRoundTrip(t *testing.T) {
	original := ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: "call_123",
		ToolName:   "bash",
		Content: []ContentBlock{
			TextContent{Type: "text", Text: "file1.txt\nfile2.txt"},
		},
		Details:   map[string]any{"exitCode": 0},
		IsError:   false,
		Timestamp: 1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ToolResultMessage
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "toolResult", decoded.Role)
	assert.Equal(t, "call_123", decoded.ToolCallID)
	assert.Equal(t, "bash", decoded.ToolName)
	assert.False(t, decoded.IsError)
	assert.Equal(t, int64(1710000000000), decoded.Timestamp)
	assert.NotNil(t, decoded.Details)
}

func TestToolResultMessageOmitEmpty(t *testing.T) {
	original := ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: "call_1",
		ToolName:   "read",
		Content: []ContentBlock{
			TextContent{Type: "text", Text: "ok"},
		},
		IsError:   false,
		Timestamp: 1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "details")
}

func TestToolResultMessageWithError(t *testing.T) {
	original := ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: "call_bad",
		ToolName:   "bash",
		Content: []ContentBlock{
			TextContent{Type: "text", Text: "command failed"},
		},
		Details:   map[string]any{"exitCode": 1},
		IsError:   true,
		Timestamp: 1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ToolResultMessage
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.True(t, decoded.IsError)
	assert.Contains(t, string(data), `"isError":true`)
}

func TestMessageSliceTypeSwitch(t *testing.T) {
	messages := []Message{
		UserMessage{
			Role:      "user",
			Content:   []ContentBlock{TextContent{Type: "text", Text: "hi"}},
			Timestamp: 1,
		},
		AssistantMessage{
			Role:       "assistant",
			Content:    []ContentBlock{TextContent{Type: "text", Text: "hello"}},
			API:        "anthropic-messages",
			Provider:   "anthropic",
			Model:      "claude-sonnet-4-20250514",
			StopReason: StopReasonStop,
			Timestamp:  2,
		},
		ToolResultMessage{
			Role:       "toolResult",
			ToolCallID: "c1",
			ToolName:   "bash",
			Content:    []ContentBlock{TextContent{Type: "text", Text: "output"}},
			Timestamp:  3,
		},
	}

	assert.Len(t, messages, 3)

	for i, msg := range messages {
		switch m := msg.(type) {
		case UserMessage:
			assert.Equal(t, "user", m.Role)
			assert.Equal(t, 0, i)
		case AssistantMessage:
			assert.Equal(t, "assistant", m.Role)
			assert.Equal(t, 1, i)
		case ToolResultMessage:
			assert.Equal(t, "toolResult", m.Role)
			assert.Equal(t, 2, i)
		default:
			t.Fatalf("unexpected type: %T", m)
		}
	}
}

func TestEmptyContentSliceNotSerializedAsNull(t *testing.T) {
	original := UserMessage{
		Role:      "user",
		Content:   []ContentBlock{},
		Timestamp: 1710000000000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"content":[]`)
	assert.NotContains(t, string(data), `"content":null`)
}
