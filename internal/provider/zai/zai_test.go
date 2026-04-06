package zai

import (
	"os"
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAPIKeyFromOptions(t *testing.T) {
	key := "test-key"
	opts := &provider.StreamOptions{APIKey: &key}
	result := ResolveAPIKey(opts)
	require.NotNil(t, result)
	assert.Equal(t, "test-key", *result)
}

func TestResolveAPIKeyFromEnv(t *testing.T) {
	t.Setenv(EnvAPIKey, "env-key")

	result := ResolveAPIKey(nil)
	require.NotNil(t, result)
	assert.Equal(t, "env-key", *result)
}

func TestResolveAPIKeyOptionsPrecedence(t *testing.T) {
	t.Setenv(EnvAPIKey, "env-key")

	key := "opts-key"
	opts := &provider.StreamOptions{APIKey: &key}
	result := ResolveAPIKey(opts)
	require.NotNil(t, result)
	assert.Equal(t, "opts-key", *result)
}

func TestResolveAPIKeyNil(t *testing.T) {
	os.Unsetenv(EnvAPIKey)
	result := ResolveAPIKey(nil)
	assert.Nil(t, result)
}

func TestBuildParamsBasic(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
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

	assert.Equal(t, "glm-5.1", params.Model)
	require.NotEmpty(t, params.Messages)
	assert.NotNil(t, params.Messages[0].OfSystem)
	assert.Nil(t, params.Messages[0].OfDeveloper)
}

func TestBuildParamsNoSystemPrompt(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
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
		ID:       "glm-5.1",
		Provider: "zai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	params := buildParams(model, ctx, nil)

	assert.Empty(t, params.Messages)
}

func TestBuildParamsNoStore(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	params := buildParams(model, ctx, nil)

	assert.False(t, params.Store.Valid())
}

func TestBuildParamsWithTools(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
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
		ID:       "glm-5.1",
		Provider: "zai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}
	temp := 0.7
	opts := &provider.StreamOptions{Temperature: &temp}

	params := buildParams(model, ctx, opts)

	assert.True(t, params.Temperature.Valid())
	assert.Equal(t, 0.7, params.Temperature.Value)
}

func TestBuildParamsWithMaxTokens(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}
	maxTokens := 4096
	opts := &provider.StreamOptions{MaxTokens: &maxTokens}

	params := buildParams(model, ctx, opts)

	assert.True(t, params.MaxCompletionTokens.Valid())
	assert.Equal(t, int64(4096), params.MaxCompletionTokens.Value)
}

func TestBuildParamsFullConversation(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
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

	assert.Equal(t, "glm-5.1", params.Model)
	require.Len(t, params.Messages, 4)
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

func TestConvertMessagesDelegates(t *testing.T) {
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
	assert.NotNil(t, params[0].OfUser)
}

func TestConvertToolsDelegates(t *testing.T) {
	tools := []provider.Tool{
		{Name: "bash", Description: "Run a command", Parameters: []byte(`{"type":"object"}`)},
	}

	params := ConvertTools(tools)

	require.Len(t, params, 1)
	assert.Equal(t, "bash", params[0].Function.Name)
}

func TestStreamReturnsEventStream(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
		BaseURL:  DefaultBaseURL,
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
		ID:       "glm-5.1",
		Provider: "zai",
		BaseURL:  DefaultBaseURL,
	}
	ctx := provider.Context{
		Messages: []message.Message{},
	}

	stream := StreamSimple(model, ctx, nil)

	require.NotNil(t, stream)
}

func TestDefaultBaseURL(t *testing.T) {
	assert.Equal(t, "https://api.z.ai/api/coding/paas/v4", DefaultBaseURL)
}

func TestNewClientUsesDefaultBaseURL(t *testing.T) {
	model := provider.Model{
		ID:       "glm-5.1",
		Provider: "zai",
	}
	key := "test-key"
	opts := &provider.StreamOptions{APIKey: &key}

	client := newClient(model, opts)

	assert.NotNil(t, client)
}
