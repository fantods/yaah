package openai

import (
	"fmt"
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildParamsBasic(t *testing.T) {
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

	params := buildParams(model, ctx, nil)

	assert.Equal(t, "gpt-4o", params.Model)
	require.NotEmpty(t, params.Messages)
	assert.NotNil(t, params.Messages[0].OfSystem)
}

func TestBuildParamsWithDeveloperRole(t *testing.T) {
	model := provider.Model{
		ID:        "o3",
		Provider:  "openai",
		Reasoning: true,
	}
	ctx := provider.Context{
		SystemPrompt: "Think step by step.",
		Messages:     []message.Message{},
	}

	params := buildParams(model, ctx, nil)

	require.NotEmpty(t, params.Messages)
	assert.NotNil(t, params.Messages[0].OfDeveloper)
}

func TestBuildParamsWithoutDeveloperRole(t *testing.T) {
	model := provider.Model{
		ID:        "gpt-4o",
		Provider:  "openai",
		Reasoning: false,
	}
	ctx := provider.Context{
		SystemPrompt: "You are helpful.",
		Messages:     []message.Message{},
	}

	params := buildParams(model, ctx, nil)

	require.NotEmpty(t, params.Messages)
	assert.NotNil(t, params.Messages[0].OfSystem)
	assert.Nil(t, params.Messages[0].OfDeveloper)
}

func TestBuildParamsWithTools(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Tools: []provider.Tool{
			{Name: "read_file", Description: "Read a file", Parameters: []byte(`{"type":"object"}`)},
		},
	}

	params := buildParams(model, ctx, nil)

	require.Len(t, params.Tools, 1)
	assert.Equal(t, "read_file", params.Tools[0].Function.Name)
}

func TestBuildParamsWithTemperature(t *testing.T) {
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

	params := buildParams(model, ctx, opts)

	assert.True(t, params.Temperature.Valid())
	assert.Equal(t, 0.7, params.Temperature.Value)
}

func TestBuildParamsWithMaxTokens(t *testing.T) {
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

	params := buildParams(model, ctx, opts)

	assert.True(t, params.MaxCompletionTokens.Valid())
	assert.Equal(t, int64(4096), params.MaxCompletionTokens.Value)
}

func TestBuildParamsWithLegacyMaxTokens(t *testing.T) {
	model := provider.Model{
		ID:       "some-model",
		Provider: "openai",
		BaseURL:  "https://chutes.ai/v1",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}
	maxTokens := 2048
	opts := &provider.StreamOptions{
		MaxTokens: &maxTokens,
	}

	params := buildParams(model, ctx, opts)

	assert.False(t, params.MaxCompletionTokens.Valid())
	assert.True(t, params.MaxTokens.Valid())
	assert.Equal(t, int64(2048), params.MaxTokens.Value)
}

func TestBuildParamsStreamOptions(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	params := buildParams(model, ctx, nil)

	assert.True(t, params.StreamOptions.IncludeUsage.Valid())
	assert.True(t, params.StreamOptions.IncludeUsage.Value)
}

func TestBuildParamsStoreDisabled(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	params := buildParams(model, ctx, nil)

	assert.True(t, params.Store.Valid())
	assert.False(t, params.Store.Value)
}

func TestDefaultCompat(t *testing.T) {
	compat := DefaultCompat()

	assert.True(t, compat.SupportsStore)
	assert.True(t, compat.SupportsDeveloperRole)
	assert.True(t, compat.SupportsReasoningEffort)
	assert.True(t, compat.SupportsUsageInStreaming)
	assert.True(t, compat.SupportsStrictMode)
	assert.Equal(t, MaxTokensFieldCompletion, compat.MaxTokensField)
	assert.Equal(t, ThinkingFormatOpenAI, compat.ThinkingFormat)
}

func TestDetectCompatOpenAI(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
	}

	compat := DetectCompat(model)

	assert.True(t, compat.SupportsStore)
	assert.True(t, compat.SupportsDeveloperRole)
	assert.True(t, compat.SupportsReasoningEffort)
}

func TestDetectCompatZAI(t *testing.T) {
	model := provider.Model{
		ID:       "model-1",
		Provider: "zai",
	}

	compat := DetectCompat(model)

	assert.False(t, compat.SupportsDeveloperRole)
	assert.False(t, compat.SupportsStore)
	assert.False(t, compat.SupportsReasoningEffort)
	assert.Equal(t, ThinkingFormatZAI, compat.ThinkingFormat)
}

func TestDetectCompatZAIByURL(t *testing.T) {
	model := provider.Model{
		ID:       "model-1",
		Provider: "custom",
		BaseURL:  "https://api.z.ai/v1",
	}

	compat := DetectCompat(model)

	assert.False(t, compat.SupportsDeveloperRole)
	assert.Equal(t, ThinkingFormatZAI, compat.ThinkingFormat)
}

func TestDetectCompatCerebras(t *testing.T) {
	model := provider.Model{
		ID:       "llama3",
		Provider: "cerebras",
		BaseURL:  "https://api.cerebras.ai/v1",
	}

	compat := DetectCompat(model)

	assert.False(t, compat.SupportsStore)
	assert.False(t, compat.SupportsDeveloperRole)
}

func TestDetectCompatXAI(t *testing.T) {
	model := provider.Model{
		ID:       "grok-3",
		Provider: "xai",
		BaseURL:  "https://api.x.ai/v1",
	}

	compat := DetectCompat(model)

	assert.False(t, compat.SupportsReasoningEffort)
}

func TestDetectCompatChutes(t *testing.T) {
	model := provider.Model{
		ID:       "model-1",
		Provider: "chutes",
		BaseURL:  "https://chutes.ai/v1",
	}

	compat := DetectCompat(model)

	assert.Equal(t, MaxTokensFieldLegacy, compat.MaxTokensField)
}

func TestDetectCompatGroq(t *testing.T) {
	model := provider.Model{
		ID:       "qwen3-32b",
		Provider: "groq",
		BaseURL:  "https://api.groq.com/openai/v1",
	}

	compat := DetectCompat(model)

	assert.False(t, compat.SupportsStore)
	assert.False(t, compat.SupportsDeveloperRole)
	require.NotNil(t, compat.ReasoningEffortMap)
	assert.Equal(t, "default", compat.ReasoningEffortMap[provider.ThinkingLevelHigh])
}

func TestDetectCompatOpenRouter(t *testing.T) {
	model := provider.Model{
		ID:       "anthropic/claude-3",
		Provider: "openrouter",
		BaseURL:  "https://openrouter.ai/api/v1",
	}

	compat := DetectCompat(model)

	assert.Equal(t, ThinkingFormatOpenRouter, compat.ThinkingFormat)
}

func TestDetectCompatCustomCompat(t *testing.T) {
	customCompat := DefaultCompat()
	customCompat.SupportsStore = false
	customCompat.MaxTokensField = MaxTokensFieldLegacy

	model := provider.Model{
		ID:       "custom-model",
		Provider: "custom",
		Compat:   customCompat,
	}

	compat := DetectCompat(model)

	assert.False(t, compat.SupportsStore)
	assert.Equal(t, MaxTokensFieldLegacy, compat.MaxTokensField)
}

func TestBuildParamsNoSystemPrompt(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{
			message.UserMessage{
				Role: "user",
				Content: []message.ContentBlock{
					message.TextContent{Type: "text", Text: "hi"},
				},
			},
		},
	}

	params := buildParams(model, ctx, nil)

	assert.NotNil(t, params.Messages[0].OfUser)
}

func TestBuildParamsEmptyMessages(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	params := buildParams(model, ctx, nil)

	assert.Empty(t, params.Messages)
}

func TestBuildParamsFullConversation(t *testing.T) {
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
						ID:        "call_1",
						Name:      "read_file",
						Arguments: map[string]any{"path": "foo.go"},
					},
				},
			},
			message.ToolResultMessage{
				Role:       "toolResult",
				ToolCallID: "call_1",
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
	}

	params := buildParams(model, ctx, opts)

	assert.Equal(t, "gpt-4o", params.Model)
	require.Len(t, params.Messages, 4) // system + user + assistant + tool
	assert.NotNil(t, params.Messages[0].OfSystem)
	assert.NotNil(t, params.Messages[1].OfUser)
	assert.NotNil(t, params.Messages[2].OfAssistant)
	assert.NotNil(t, params.Messages[3].OfTool)
	require.Len(t, params.Tools, 1)
	assert.True(t, params.Temperature.Valid())
	assert.Equal(t, 0.5, params.Temperature.Value)
	assert.True(t, params.MaxCompletionTokens.Valid())
	assert.Equal(t, int64(8192), params.MaxCompletionTokens.Value)
}

func TestStreamReturnsEventStream(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	stream := Stream(model, ctx, nil)
	require.NotNil(t, stream)
	require.NotNil(t, stream.Events())
}

func TestStreamSimpleReturnsEventStream(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4o",
		Provider: "openai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	stream := StreamSimple(model, ctx, nil)
	require.NotNil(t, stream)
}

func TestFormatOpenAIError(t *testing.T) {
	err := formatOpenAIError(fmt.Errorf("something went wrong"))
	assert.Contains(t, err, "openai:")
	assert.Contains(t, err, "something went wrong")
}
