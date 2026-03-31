package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fantods/yaah/internal/message"
)

func TestContextConstruction(t *testing.T) {
	ctx := Context{
		SystemPrompt: "You are helpful.",
		Messages: []message.Message{
			message.UserMessage{
				Role:      "user",
				Content:   []message.ContentBlock{message.TextContent{Type: "text", Text: "hi"}},
				Timestamp: 1,
			},
		},
		Tools: []Tool{
			{Name: "bash", Description: "Run commands", Parameters: json.RawMessage(`{"type":"object"}`)},
		},
	}

	assert.Equal(t, "You are helpful.", ctx.SystemPrompt)
	assert.Len(t, ctx.Messages, 1)
	assert.Len(t, ctx.Tools, 1)
	assert.Equal(t, "bash", ctx.Tools[0].Name)
}

func TestContextJSONRoundTrip(t *testing.T) {
	original := Context{
		SystemPrompt: "Be concise.",
		Messages: []message.Message{
			message.UserMessage{
				Role:      "user",
				Content:   []message.ContentBlock{message.TextContent{Type: "text", Text: "hello"}},
				Timestamp: 100,
			},
			message.AssistantMessage{
				Role:       "assistant",
				Content:    []message.ContentBlock{message.TextContent{Type: "text", Text: "hi"}},
				API:        "anthropic-messages",
				Provider:   "anthropic",
				Model:      "claude-sonnet-4-20250514",
				StopReason: message.StopReasonStop,
				Timestamp:  200,
			},
		},
		Tools: []Tool{
			{Name: "read", Description: "Read files", Parameters: json.RawMessage(`{"type":"object"}`)},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Context
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "Be concise.", decoded.SystemPrompt)
	assert.Len(t, decoded.Messages, 2)
	assert.Len(t, decoded.Tools, 1)
	assert.Equal(t, "read", decoded.Tools[0].Name)
}

func TestContextEmpty(t *testing.T) {
	ctx := Context{}
	assert.Equal(t, "", ctx.SystemPrompt)
	assert.Nil(t, ctx.Messages)
	assert.Nil(t, ctx.Tools)
}

func TestContextOmitEmpty(t *testing.T) {
	ctx := Context{
		Messages: []message.Message{},
		Tools:    []Tool{},
	}

	data, err := json.Marshal(ctx)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "systemPrompt")
}
