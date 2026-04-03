package openai

import (
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildResponsesParamsBasic(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		SystemPrompt: "You are helpful.",
		Messages: []message.Message{
			message.UserMessage{
				Role: "user",
				Content: []message.ContentBlock{
					message.TextContent{Type: "text", Text: "hello"},
				},
			},
		},
	}

	params := buildResponsesParams(model, ctx, nil)

	assert.Equal(t, "gpt-4o", string(params.Model))
	assert.True(t, params.Instructions.Valid())
	assert.Equal(t, "You are helpful.", params.Instructions.Value)
	require.NotNil(t, params.Input.OfInputItemList)
	assert.True(t, params.Store.Valid())
	assert.False(t, params.Store.Value)
}

func TestBuildResponsesParamsNoSystemPrompt(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	params := buildResponsesParams(model, ctx, nil)

	assert.False(t, params.Instructions.Valid())
}

func TestBuildResponsesParamsWithTools(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
		Tools: []provider.Tool{
			{Name: "read_file", Description: "Read a file", Parameters: []byte(`{"type":"object"}`)},
		},
	}

	params := buildResponsesParams(model, ctx, nil)

	require.Len(t, params.Tools, 1)
	assert.Equal(t, "read_file", params.Tools[0].OfFunction.Name)
}

func TestBuildResponsesParamsWithTemperature(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}
	temp := 0.7
	opts := &provider.StreamOptions{
		Temperature: &temp,
	}

	params := buildResponsesParams(model, ctx, opts)

	assert.True(t, params.Temperature.Valid())
	assert.Equal(t, 0.7, params.Temperature.Value)
}

func TestBuildResponsesParamsWithMaxTokens(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}
	maxTokens := 4096
	opts := &provider.StreamOptions{
		MaxTokens: &maxTokens,
	}

	params := buildResponsesParams(model, ctx, opts)

	assert.True(t, params.MaxOutputTokens.Valid())
	assert.Equal(t, int64(4096), params.MaxOutputTokens.Value)
}

func TestBuildResponsesParamsWithReasoning(t *testing.T) {
	model := provider.Model{
		ID:        "o3",
		Provider:  "openai",
		Reasoning: true,
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	params := buildResponsesParams(model, ctx, nil)

	assert.Equal(t, "medium", string(params.Reasoning.Effort))
	require.Len(t, params.Include, 1)
}

func TestBuildResponsesParamsWithSessionID(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}
	opts := &provider.StreamOptions{
		SessionID: "sess_abc123",
	}

	params := buildResponsesParams(model, ctx, opts)

	assert.True(t, params.PromptCacheKey.Valid())
	assert.Equal(t, "sess_abc123", params.PromptCacheKey.Value)
}

func TestConvertUserMessageResponsesText(t *testing.T) {
	m := message.UserMessage{
		Role: "user",
		Content: []message.ContentBlock{
			message.TextContent{Type: "text", Text: "hello"},
		},
	}

	items := convertUserMessageForResponses(m)
	require.Len(t, items, 1)
	assert.NotNil(t, items[0].OfMessage)
}

func TestConvertUserMessageResponsesImage(t *testing.T) {
	m := message.UserMessage{
		Role: "user",
		Content: []message.ContentBlock{
			message.TextContent{Type: "text", Text: "look at this"},
			message.ImageContent{Type: "image", Data: "aGVsbG8=", MIMEType: "image/png"},
		},
	}

	items := convertUserMessageForResponses(m)
	require.Len(t, items, 1)
	assert.NotNil(t, items[0].OfInputMessage)
}

func TestConvertUserMessageResponsesEmpty(t *testing.T) {
	m := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{},
	}

	items := convertUserMessageForResponses(m)
	assert.Nil(t, items)
}

func TestConvertAssistantMessageResponsesText(t *testing.T) {
	m := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.TextContent{Type: "text", Text: "hi there"},
		},
	}

	items := convertAssistantMessageForResponses(m)
	require.Len(t, items, 1)
	assert.NotNil(t, items[0].OfMessage)
}

func TestConvertAssistantMessageResponsesToolCall(t *testing.T) {
	m := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.ToolCall{
				Type:      "toolCall",
				ID:        "call_abc|fc_123",
				Name:      "read_file",
				Arguments: map[string]any{"path": "/tmp/x"},
			},
		},
	}

	items := convertAssistantMessageForResponses(m)
	require.Len(t, items, 1)
	assert.NotNil(t, items[0].OfFunctionCall)
	assert.Equal(t, "call_abc", items[0].OfFunctionCall.CallID)
	assert.Equal(t, "read_file", items[0].OfFunctionCall.Name)
	assert.True(t, items[0].OfFunctionCall.ID.Valid())
	assert.Equal(t, "fc_123", items[0].OfFunctionCall.ID.Value)
}

func TestConvertAssistantMessageResponsesThinking(t *testing.T) {
	m := message.AssistantMessage{
		Role: "assistant",
		Content: []message.ContentBlock{
			message.ThinkingContent{
				Type:              "thinking",
				Thinking:          "let me think",
				ThinkingSignature: `{"id":"rs_abc","summary":[],"type":"reasoning"}`,
			},
		},
	}

	items := convertAssistantMessageForResponses(m)
	require.Len(t, items, 1)
	assert.NotNil(t, items[0].OfReasoning)
}

func TestConvertToolResultMessageResponses(t *testing.T) {
	m := message.ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: "call_abc|fc_123",
		ToolName:   "read_file",
		Content: []message.ContentBlock{
			message.TextContent{Type: "text", Text: "file contents here"},
		},
	}

	items := convertToolResultMessageForResponses(m)
	require.Len(t, items, 1)
	assert.NotNil(t, items[0].OfFunctionCallOutput)
	assert.Equal(t, "call_abc", items[0].OfFunctionCallOutput.CallID)
	assert.Equal(t, "file contents here", items[0].OfFunctionCallOutput.Output)
}

func TestConvertToolResultMessageResponsesEmpty(t *testing.T) {
	m := message.ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: "call_abc",
		Content:    []message.ContentBlock{},
	}

	items := convertToolResultMessageForResponses(m)
	require.Len(t, items, 1)
	assert.Equal(t, "(no output)", items[0].OfFunctionCallOutput.Output)
}

func TestConvertResponsesTools(t *testing.T) {
	tools := []provider.Tool{
		{Name: "read_file", Description: "Read a file", Parameters: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`)},
	}

	params := convertResponsesTools(tools)
	require.Len(t, params, 1)
	assert.Equal(t, "read_file", params[0].OfFunction.Name)
	assert.True(t, params[0].OfFunction.Description.Valid())
	assert.Equal(t, "Read a file", params[0].OfFunction.Description.Value)
}

func TestMapResponseStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected message.StopReason
	}{
		{"completed", message.StopReasonStop},
		{"incomplete", message.StopReasonLength},
		{"failed", message.StopReasonError},
		{"cancelled", message.StopReasonError},
		{"in_progress", message.StopReasonStop},
		{"queued", message.StopReasonStop},
		{"unknown", message.StopReasonError},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapResponseStatus(tt.input))
		})
	}
}

func TestParseStreamingJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]any
	}{
		{"empty", "", nil},
		{"valid", `{"path":"/tmp/x"}`, map[string]any{"path": "/tmp/x"}},
		{"invalid", `{broken`, nil},
		{"partial", `{"path":"/tmp`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStreamingJSON(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStreamResponsesReturnsEventStream(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	stream := StreamResponses(model, ctx, nil)
	require.NotNil(t, stream)
	require.NotNil(t, stream.Events())
}

func TestStreamSimpleResponsesReturnsEventStream(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	stream := StreamSimpleResponses(model, ctx, nil)
	require.NotNil(t, stream)
}

func TestBuildResponsesParamsFullConversation(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		SystemPrompt: "You are a coding assistant.",
		Messages: []message.Message{
			message.UserMessage{
				Role: "user",
				Content: []message.ContentBlock{
					message.TextContent{Type: "text", Text: "read foo.go"},
				},
			},
			message.AssistantMessage{
				Role: "assistant",
				Content: []message.ContentBlock{
					message.ToolCall{
						Type:      "toolCall",
						ID:        "call_1|fc_item1",
						Name:      "read_file",
						Arguments: map[string]any{"path": "foo.go"},
					},
				},
			},
			message.ToolResultMessage{
				Role:       "toolResult",
				ToolCallID: "call_1|fc_item1",
				ToolName:   "read_file",
				Content: []message.ContentBlock{
					message.TextContent{Type: "text", Text: "package main"},
				},
			},
		},
		Tools: []provider.Tool{
			{Name: "read_file", Description: "Read a file", Parameters: []byte(`{"type":"object"}`)},
		},
	}
	temp := 0.5
	maxTokens := 8192
	opts := &provider.StreamOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		SessionID:   "sess_123",
	}

	params := buildResponsesParams(model, ctx, opts)

	assert.Equal(t, "gpt-4o", string(params.Model))
	assert.True(t, params.Instructions.Valid())
	assert.True(t, params.Temperature.Valid())
	assert.True(t, params.MaxOutputTokens.Valid())
	assert.True(t, params.PromptCacheKey.Valid())
	require.Len(t, params.Tools, 1)
	require.NotNil(t, params.Input.OfInputItemList)
}
