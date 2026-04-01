package anthropic

import (
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/require"
)

func intPtr(v int) *int { return &v }

func TestStreamReturnsEventStream(t *testing.T) {
	model := provider.Model{
		ID:        "claude-3-5-sonnet",
		Provider:  "anthropic",
		MaxTokens: 8192,
	}

	ctx := provider.Context{
		SystemPrompt: "You are helpful.",
		Messages: []message.Message{
			message.UserMessage{
				Role:      "user",
				Content:   []message.ContentBlock{message.TextContent{Type: "text", Text: "hi"}},
				Timestamp: 1000,
			},
		},
	}

	apiKey := "sk-test-key"
	opts := &provider.StreamOptions{
		APIKey:    &apiKey,
		MaxTokens: intPtr(4096),
	}

	stream := Stream(model, ctx, opts)
	require.NotNil(t, stream)
}

func TestStreamSimpleReturnsEventStream(t *testing.T) {
	model := provider.Model{
		ID:        "claude-3-5-sonnet",
		Provider:  "anthropic",
		MaxTokens: 8192,
	}

	ctx := provider.Context{
		SystemPrompt: "You are helpful.",
		Messages: []message.Message{
			message.UserMessage{
				Role:      "user",
				Content:   []message.ContentBlock{message.TextContent{Type: "text", Text: "hi"}},
				Timestamp: 1000,
			},
		},
	}

	apiKey := "sk-test-key"
	opts := &provider.StreamOptions{
		APIKey:    &apiKey,
		MaxTokens: intPtr(4096),
	}

	stream := StreamSimple(model, ctx, opts)
	require.NotNil(t, stream)
}

func TestStreamWithEmptyMessages(t *testing.T) {
	model := provider.Model{
		ID:        "claude-3-5-sonnet",
		Provider:  "anthropic",
		MaxTokens: 8192,
	}

	ctx := provider.Context{
		SystemPrompt: "You are helpful.",
		Messages:     []message.Message{},
	}

	apiKey := "sk-test-key"
	opts := &provider.StreamOptions{
		APIKey:    &apiKey,
		MaxTokens: intPtr(4096),
	}

	stream := Stream(model, ctx, opts)
	require.NotNil(t, stream)
}

func TestStreamWithTools(t *testing.T) {
	model := provider.Model{
		ID:        "claude-3-5-sonnet",
		Provider:  "anthropic",
		MaxTokens: 8192,
	}

	ctx := provider.Context{
		SystemPrompt: "You are helpful.",
		Messages: []message.Message{
			message.UserMessage{
				Role:      "user",
				Content:   []message.ContentBlock{message.TextContent{Type: "text", Text: "read foo.txt"}},
				Timestamp: 1000,
			},
		},
		Tools: []provider.Tool{
			{Name: "read_file", Description: "Read a file", Parameters: []byte(`{"type":"object"}`)},
		},
	}

	apiKey := "sk-test-key"
	opts := &provider.StreamOptions{
		APIKey:    &apiKey,
		MaxTokens: intPtr(4096),
	}

	stream := Stream(model, ctx, opts)
	require.NotNil(t, stream)
}

func TestStreamWithCustomBaseURL(t *testing.T) {
	model := provider.Model{
		ID:        "custom-model",
		Provider:  "anthropic",
		BaseURL:   "https://custom-api.example.com",
		MaxTokens: 4096,
	}

	ctx := provider.Context{
		SystemPrompt: "test",
		Messages: []message.Message{
			message.UserMessage{
				Role:      "user",
				Content:   []message.ContentBlock{message.TextContent{Type: "text", Text: "hi"}},
				Timestamp: 1000,
			},
		},
	}

	apiKey := "sk-test"
	opts := &provider.StreamOptions{
		APIKey:    &apiKey,
		MaxTokens: intPtr(4096),
		Headers:   map[string]string{"X-Request": "req-val"},
	}

	stream := Stream(model, ctx, opts)
	require.NotNil(t, stream)
}
