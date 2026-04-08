package openai

import (
	"encoding/json"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/openai/openai-go"
)

// ChunkParser maintains state for parsing OpenAI streaming chunks.
type ChunkParser struct {
	partial *message.AssistantMessage
	model   provider.Model

	// Current block state
	currentBlockType string // "text", "thinking", "toolCall"
	currentIndex     int64  // Current content block index
	currentToolIndex int64  // For tool calls, the delta index

	// JSON parsers for tool call arguments (keyed by tool call index)
	jsonParsers map[int64]*provider.PartialJSONParser
}

// NewChunkParser creates a new parser with an initialized partial message.
func NewChunkParser(model provider.Model) *ChunkParser {
	return &ChunkParser{
		model: model,
		partial: &message.AssistantMessage{
			Role:     "assistant",
			API:      "openai-completions",
			Provider: model.Provider,
			Model:    model.ID,
			Content:  []message.ContentBlock{},
			Usage: message.Usage{
				Cost: message.Cost{},
			},
			StopReason: message.StopReasonStop,
		},
		jsonParsers: make(map[int64]*provider.PartialJSONParser),
	}
}

// ParseChunk parses a single chunk and returns events.
// Returns nil events slice if chunk should be skipped.
func (p *ChunkParser) ParseChunk(chunk openai.ChatCompletionChunk) []provider.AssistantMessageEvent {
	var events []provider.AssistantMessageEvent

	// Capture response ID (same for all chunks)
	if chunk.ID != "" && p.partial.ResponseID == "" {
		p.partial.ResponseID = chunk.ID
	}

	// Parse usage if present (usually in final chunk with stream_options.include_usage)
	if chunk.JSON.Usage.Valid() {
		p.parseUsage(chunk.Usage)
	}

	// Skip if no choices
	if len(chunk.Choices) == 0 {
		return events
	}

	choice := chunk.Choices[0]

	// Handle finish reason
	if choice.FinishReason != "" {
		p.mapStopReason(choice.FinishReason)
	}

	// Parse delta content
	events = append(events, p.parseDelta(choice.Delta)...)

	return events
}

// parseDelta parses the delta content and returns events.
func (p *ChunkParser) parseDelta(delta openai.ChatCompletionChunkChoiceDelta) []provider.AssistantMessageEvent {
	var events []provider.AssistantMessageEvent

	// 1. Text content
	if delta.Content != "" {
		events = append(events, p.handleTextDelta(delta.Content)...)
	}

	// 2. Tool calls
	for _, tc := range delta.ToolCalls {
		events = append(events, p.handleToolCallDelta(tc)...)
	}

	return events
}

// ParseReasoningDelta handles a reasoning content delta and returns events.
func (p *ChunkParser) ParseReasoningDelta(content string) []provider.AssistantMessageEvent {
	if content == "" {
		return nil
	}

	var events []provider.AssistantMessageEvent

	if p.currentBlockType != "thinking" {
		p.finishCurrentBlock()
		p.currentBlockType = "thinking"
		p.partial.Content = append(p.partial.Content, message.ThinkingContent{
			Type:     "thinking",
			Thinking: "",
		})
		p.currentIndex = int64(len(p.partial.Content) - 1)

		events = append(events, provider.EventThinkingStart{
			Partial:      p.partial,
			ContentIndex: int(p.currentIndex),
		})
	}

	if tc, ok := p.partial.Content[p.currentIndex].(message.ThinkingContent); ok {
		tc.Thinking += content
		p.partial.Content[p.currentIndex] = tc
	}

	events = append(events, provider.EventThinkingDelta{
		Partial:      p.partial,
		ContentIndex: int(p.currentIndex),
		Delta:        content,
	})

	return events
}

// ParseReasoningTextDelta handles a text content delta from a raw SSE stream.
func (p *ChunkParser) ParseReasoningTextDelta(content string) []provider.AssistantMessageEvent {
	return p.handleTextDelta(content)
}

// ParseToolCallDelta handles a tool call delta from a raw SSE stream.
func (p *ChunkParser) ParseToolCallDelta(index int64, id string, name string, arguments string) []provider.AssistantMessageEvent {
	tc := openai.ChatCompletionChunkChoiceDeltaToolCall{
		Index: index,
		ID:    id,
		Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
			Name:      name,
			Arguments: arguments,
		},
	}
	return p.handleToolCallDelta(tc)
}

// SetStopReason sets the stop reason and optional error message on the partial.
func (p *ChunkParser) SetStopReason(reason message.StopReason, errMsg string) {
	p.partial.StopReason = reason
	p.partial.ErrorMessage = errMsg
}

// ParseRawUsage parses a raw JSON usage object and updates the partial.
func (p *ChunkParser) ParseRawUsage(rawJSON string) {
	var usage openai.CompletionUsage
	if json.Unmarshal([]byte(rawJSON), &usage) != nil {
		return
	}
	p.partial.Usage = ParseUsage(usage)
}

// handleTextDelta handles text content deltas.
func (p *ChunkParser) handleTextDelta(content string) []provider.AssistantMessageEvent {
	var events []provider.AssistantMessageEvent

	// Start new text block if needed
	if p.currentBlockType != "text" {
		p.finishCurrentBlock()
		p.currentBlockType = "text"
		p.partial.Content = append(p.partial.Content, message.TextContent{Type: "text"})
		p.currentIndex = int64(len(p.partial.Content) - 1)

		events = append(events, provider.EventTextStart{
			Partial:      p.partial,
			ContentIndex: int(p.currentIndex),
		})
	}

	// Append delta
	if tc, ok := p.partial.Content[p.currentIndex].(message.TextContent); ok {
		tc.Text += content
		p.partial.Content[p.currentIndex] = tc
	}

	events = append(events, provider.EventTextDelta{
		Partial:      p.partial,
		ContentIndex: int(p.currentIndex),
		Delta:        content,
	})

	return events
}

// handleToolCallDelta handles tool call deltas.
func (p *ChunkParser) handleToolCallDelta(tc openai.ChatCompletionChunkChoiceDeltaToolCall) []provider.AssistantMessageEvent {
	var events []provider.AssistantMessageEvent

	// Check if we need to start a new tool call block
	// New tool call if: different type, different index, or new ID that doesn't match
	isNewToolCall := p.currentBlockType != "toolCall" ||
		tc.ID != "" && p.getToolCallID(p.currentToolIndex) != tc.ID

	if isNewToolCall {
		p.finishCurrentBlock()
		p.currentBlockType = "toolCall"
		p.currentToolIndex = tc.Index

		// Create new tool call block
		newTC := message.ToolCall{
			Type:      "toolCall",
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: map[string]any{},
		}
		p.partial.Content = append(p.partial.Content, newTC)
		p.currentIndex = int64(len(p.partial.Content) - 1)

		// Create JSON parser for this tool call
		p.jsonParsers[tc.Index] = provider.NewPartialJSONParser()

		events = append(events, provider.EventToolCallStart{
			Partial:      p.partial,
			ContentIndex: int(p.currentIndex),
		})
	}

	// Update tool call if we have data
	if toolCall, ok := p.partial.Content[p.currentIndex].(message.ToolCall); ok {
		// Update ID if provided
		if tc.ID != "" {
			toolCall.ID = tc.ID
		}
		// Update name if provided
		if tc.Function.Name != "" {
			toolCall.Name = tc.Function.Name
		}
		// Parse arguments incrementally
		if tc.Function.Arguments != "" {
			if parser, ok := p.jsonParsers[p.currentToolIndex]; ok {
				args, _ := parser.Parse(tc.Function.Arguments)
				if args != nil {
					toolCall.Arguments = args
				}
			}
		}
		p.partial.Content[p.currentIndex] = toolCall

		events = append(events, provider.EventToolCallDelta{
			Partial:      p.partial,
			ContentIndex: int(p.currentIndex),
			Delta:        tc.Function.Arguments,
		})
	}

	return events
}

// finishCurrentBlock emits the end event for the current block.
func (p *ChunkParser) finishCurrentBlock() []provider.AssistantMessageEvent {
	if p.currentBlockType == "" || p.currentIndex < 0 || int(p.currentIndex) >= len(p.partial.Content) {
		return nil
	}

	var events []provider.AssistantMessageEvent

	switch p.currentBlockType {
	case "text":
		if tc, ok := p.partial.Content[p.currentIndex].(message.TextContent); ok {
			events = append(events, provider.EventTextEnd{
				Partial:      p.partial,
				ContentIndex: int(p.currentIndex),
				Content:      tc.Text,
			})
		}
	case "thinking":
		if tc, ok := p.partial.Content[p.currentIndex].(message.ThinkingContent); ok {
			events = append(events, provider.EventThinkingEnd{
				Partial:      p.partial,
				ContentIndex: int(p.currentIndex),
				Content:      tc.Thinking,
			})
		}
	case "toolCall":
		if tc, ok := p.partial.Content[p.currentIndex].(message.ToolCall); ok {
			events = append(events, provider.EventToolCallEnd{
				Partial:      p.partial,
				ContentIndex: int(p.currentIndex),
				ToolCall:     tc,
			})
		}
	}

	p.currentBlockType = ""
	return events
}

// Finalize finishes parsing and returns the final events.
func (p *ChunkParser) Finalize() []provider.AssistantMessageEvent {
	var events []provider.AssistantMessageEvent

	// Finish current block
	events = append(events, p.finishCurrentBlock()...)

	// Emit done event
	events = append(events, provider.EventDone{
		Reason:  p.partial.StopReason,
		Message: *p.partial,
	})

	return events
}

// FinalizeWithError finishes parsing with an error.
func (p *ChunkParser) FinalizeWithError(err error) []provider.AssistantMessageEvent {
	p.partial.StopReason = message.StopReasonError
	p.partial.ErrorMessage = err.Error()

	return []provider.AssistantMessageEvent{
		provider.EventError{
			Reason:  message.StopReasonError,
			Message: *p.partial,
		},
	}
}

// parseUsage extracts usage information from the chunk.
func (p *ChunkParser) parseUsage(usage openai.CompletionUsage) {
	cachedTokens := usage.PromptTokensDetails.CachedTokens
	reasoningTokens := usage.CompletionTokensDetails.ReasoningTokens

	// OpenAI includes cached tokens in prompt_tokens, so subtract to get non-cached input
	input := usage.PromptTokens - cachedTokens
	// Output includes reasoning tokens for total
	outputTokens := usage.CompletionTokens + reasoningTokens

	p.partial.Usage = message.Usage{
		Input:       input,
		Output:      outputTokens,
		CacheRead:   cachedTokens,
		CacheWrite:  0,
		TotalTokens: input + outputTokens,
		Cost:        message.Cost{},
	}
}

// mapStopReason maps OpenAI finish reasons to internal stop reasons.
func (p *ChunkParser) mapStopReason(reason string) {
	switch reason {
	case "stop", "end", "":
		p.partial.StopReason = message.StopReasonStop
	case "length":
		p.partial.StopReason = message.StopReasonLength
	case "tool_calls", "function_call":
		p.partial.StopReason = message.StopReasonToolUse
	case "content_filter":
		p.partial.StopReason = message.StopReasonError
		p.partial.ErrorMessage = "Provider finish_reason: content_filter"
	case "network_error":
		p.partial.StopReason = message.StopReasonError
		p.partial.ErrorMessage = "Provider finish_reason: network_error"
	default:
		p.partial.StopReason = message.StopReasonError
		p.partial.ErrorMessage = "Provider finish_reason: " + reason
	}
}

// getToolCallID gets the ID of a tool call by tool delta index.
func (p *ChunkParser) getToolCallID(toolIndex int64) string {
	for i, block := range p.partial.Content {
		if tc, ok := block.(message.ToolCall); ok {
			// Match by stored tool index if we track it, otherwise by content position
			if int64(i) == p.currentIndex && p.currentBlockType == "toolCall" {
				return tc.ID
			}
			// Also check if we have a matching tool index
			if parser, ok := p.jsonParsers[toolIndex]; ok && parser != nil {
				return tc.ID
			}
		}
	}
	return ""
}

// Partial returns the current partial message.
func (p *ChunkParser) Partial() *message.AssistantMessage {
	return p.partial
}

// SetTimestamp sets the timestamp on the partial message.
func (p *ChunkParser) SetTimestamp(ts int64) {
	p.partial.Timestamp = ts
}

// ParseUsage is a helper to parse usage from a completion usage struct.
// This is exposed for testing and for usage in the stream function.
func ParseUsage(usage openai.CompletionUsage) message.Usage {
	cachedTokens := usage.PromptTokensDetails.CachedTokens
	reasoningTokens := usage.CompletionTokensDetails.ReasoningTokens

	input := usage.PromptTokens - cachedTokens
	outputTokens := usage.CompletionTokens + reasoningTokens

	return message.Usage{
		Input:       input,
		Output:      outputTokens,
		CacheRead:   cachedTokens,
		CacheWrite:  0,
		TotalTokens: input + outputTokens,
		Cost:        message.Cost{},
	}
}

// MapStopReason is a helper to map finish reasons.
// This is exposed for testing.
func MapStopReason(reason string) (message.StopReason, string) {
	switch reason {
	case "stop", "end", "":
		return message.StopReasonStop, ""
	case "length":
		return message.StopReasonLength, ""
	case "tool_calls", "function_call":
		return message.StopReasonToolUse, ""
	case "content_filter":
		return message.StopReasonError, "Provider finish_reason: content_filter"
	case "network_error":
		return message.StopReasonError, "Provider finish_reason: network_error"
	default:
		return message.StopReasonError, "Provider finish_reason: " + reason
	}
}
