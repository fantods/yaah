package openai

import (
	"testing"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func TestParseChunkText(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	parser := NewChunkParser(model)

	// Start event
	startEvents := parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					Content: "Hello",
				},
				Index: 0,
			},
		},
	})

	require.Len(t, startEvents, 2)
	_, isTextStart := startEvents[0].(provider.EventTextStart)
	assert.True(t, isTextStart, "Expected EventTextStart")
	textDelta, isTextDelta := startEvents[1].(provider.EventTextDelta)
	assert.True(t, isTextDelta, "Expected EventTextDelta")
	assert.Equal(t, "Hello", textDelta.Delta)

	// Continue streaming
	continueEvents := parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					Content: " world",
				},
				Index: 0,
			},
		},
	})

	require.Len(t, continueEvents, 1)
	textDelta2, isTextDelta := continueEvents[0].(provider.EventTextDelta)
	assert.True(t, isTextDelta, "Expected EventTextDelta")
	assert.Equal(t, " world", textDelta2.Delta)

	// Check partial message
	partial := parser.Partial()
	assert.Equal(t, "chatcmpl-123", partial.ResponseID)
	require.Len(t, partial.Content, 1)
	textContent, ok := partial.Content[0].(message.TextContent)
	require.True(t, ok)
	assert.Equal(t, "Hello world", textContent.Text)
}

func TestParseChunkToolCall(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	parser := NewChunkParser(model)

	// First chunk: tool call starts with ID and name
	events1 := parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					ToolCalls: []openai.ChatCompletionChunkChoiceDeltaToolCall{
						{
							Index: 0,
							ID:    "call_abc",
							Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
								Name: "read_file",
							},
						},
					},
				},
				Index: 0,
			},
		},
	})

	require.Len(t, events1, 2)
	_, isToolCallStart := events1[0].(provider.EventToolCallStart)
	assert.True(t, isToolCallStart, "Expected EventToolCallStart")

	// Second chunk: partial arguments
	events2 := parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					ToolCalls: []openai.ChatCompletionChunkChoiceDeltaToolCall{
						{
							Index: 0,
							Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
								Arguments: `{"path": "/tm`,
							},
						},
					},
				},
				Index: 0,
			},
		},
	})

	require.Len(t, events2, 1)
	toolCallDelta, isToolCallDelta := events2[0].(provider.EventToolCallDelta)
	assert.True(t, isToolCallDelta, "Expected EventToolCallDelta")
	assert.Equal(t, `{"path": "/tm`, toolCallDelta.Delta)

	// Third chunk: complete arguments
	events3 := parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					ToolCalls: []openai.ChatCompletionChunkChoiceDeltaToolCall{
						{
							Index: 0,
							Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
								Arguments: `p/test"}`,
							},
						},
					},
				},
				Index: 0,
			},
		},
	})

	require.Len(t, events3, 1)

	// Check partial message
	partial := parser.Partial()
	require.Len(t, partial.Content, 1)
	toolCall, ok := partial.Content[0].(message.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_abc", toolCall.ID)
	assert.Equal(t, "read_file", toolCall.Name)
	assert.Equal(t, map[string]any{"path": "/tmp/test"}, toolCall.Arguments)
}

func TestParseChunkMultipleToolCalls(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	parser := NewChunkParser(model)

	// First tool call
	_ = parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					ToolCalls: []openai.ChatCompletionChunkChoiceDeltaToolCall{
						{
							Index: 0,
							ID:    "call_1",
							Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
								Name: "read",
							},
						},
					},
				},
				Index: 0,
			},
		},
	})

	// Second tool call (different index)
	_ = parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					ToolCalls: []openai.ChatCompletionChunkChoiceDeltaToolCall{
						{
							Index: 1,
							ID:    "call_2",
							Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
								Name: "write",
							},
						},
					},
				},
				Index: 0,
			},
		},
	})

	partial := parser.Partial()
	require.Len(t, partial.Content, 2)

	tc1, ok := partial.Content[0].(message.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_1", tc1.ID)
	assert.Equal(t, "read", tc1.Name)

	tc2, ok := partial.Content[1].(message.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_2", tc2.ID)
	assert.Equal(t, "write", tc2.Name)
}

func TestParseChunkUsage(t *testing.T) {
	// Test the ParseUsage helper directly since chunk.Usage is not optional in Go SDK
	usage := ParseUsage(openai.CompletionUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		PromptTokensDetails: openai.CompletionUsagePromptTokensDetails{
			CachedTokens: 20,
		},
		CompletionTokensDetails: openai.CompletionUsageCompletionTokensDetails{
			ReasoningTokens: 10,
		},
	})

	// Cached tokens subtracted from input
	assert.Equal(t, int64(80), usage.Input)  // 100 - 20
	// Output includes reasoning tokens
	assert.Equal(t, int64(60), usage.Output) // 50 + 10
	assert.Equal(t, int64(20), usage.CacheRead)
}

func TestMapStopReason(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}

	tests := []struct {
		input    string
		expected message.StopReason
		hasError bool
	}{
		{"stop", message.StopReasonStop, false},
		{"length", message.StopReasonLength, false},
		{"tool_calls", message.StopReasonToolUse, false},
		{"function_call", message.StopReasonToolUse, false},
		{"content_filter", message.StopReasonError, true},
		{"network_error", message.StopReasonError, true},
		{"unknown", message.StopReasonError, true},
		{"", message.StopReasonStop, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parser := NewChunkParser(model)
			parser.mapStopReason(tt.input)

			assert.Equal(t, tt.expected, parser.Partial().StopReason)
			if tt.hasError {
				assert.NotEmpty(t, parser.Partial().ErrorMessage)
			} else {
				assert.Empty(t, parser.Partial().ErrorMessage)
			}
		})
	}
}

func TestParseChunkEmpty(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	parser := NewChunkParser(model)

	// Empty choices
	events := parser.ParseChunk(openai.ChatCompletionChunk{
		ID:      "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{},
	})

	assert.Empty(t, events)
}

func TestParseChunkFinishReason(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	parser := NewChunkParser(model)

	// Send text content first
	_ = parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					Content: "Done",
				},
				Index: 0,
			},
		},
	})

	// Chunk with finish reason
	_ = parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta:        openai.ChatCompletionChunkChoiceDelta{},
				FinishReason: "stop",
				Index:        0,
			},
		},
	})

	assert.Equal(t, message.StopReasonStop, parser.Partial().StopReason)
}

func TestFinalize(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	parser := NewChunkParser(model)

	// Send some content
	_ = parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					Content: "Hello",
				},
				Index: 0,
			},
		},
	})

	// Finalize
	events := parser.Finalize()

	// Should emit EventTextEnd then EventDone
	require.Len(t, events, 2)
	_, isTextEnd := events[0].(provider.EventTextEnd)
	assert.True(t, isTextEnd, "Expected EventTextEnd")

	done, isDone := events[1].(provider.EventDone)
	assert.True(t, isDone, "Expected EventDone")
	assert.Equal(t, message.StopReasonStop, done.Reason)
	assert.Equal(t, "Hello", done.Message.Content[0].(message.TextContent).Text)
}

func TestParseChunkTextThenToolCall(t *testing.T) {
	model := provider.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	parser := NewChunkParser(model)

	// Text first
	_ = parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					Content: "Let me check",
				},
				Index: 0,
			},
		},
	})

	// Then tool call
	_ = parser.ParseChunk(openai.ChatCompletionChunk{
		ID: "chatcmpl-123",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Delta: openai.ChatCompletionChunkChoiceDelta{
					ToolCalls: []openai.ChatCompletionChunkChoiceDeltaToolCall{
						{
							Index: 0,
							ID:    "call_1",
							Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
								Name: "search",
							},
						},
					},
				},
				Index: 0,
			},
		},
	})

	partial := parser.Partial()
	require.Len(t, partial.Content, 2)

	// First block is text
	tc1, ok := partial.Content[0].(message.TextContent)
	require.True(t, ok)
	assert.Equal(t, "Let me check", tc1.Text)

	// Second block is tool call
	tc2, ok := partial.Content[1].(message.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_1", tc2.ID)
}
