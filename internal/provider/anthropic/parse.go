package anthropic

import (
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func HandleEvent(
	evt anthropic.MessageStreamEventUnion,
	partial *message.AssistantMessage,
	events chan<- provider.AssistantMessageEvent,
) {
	switch evt.Type {
	case "message_start":
		handleMessageStart(evt, partial, events)
	case "content_block_start":
		handleContentBlockStart(evt, partial, events)
	case "content_block_delta":
		handleContentBlockDelta(evt, partial, events)
	case "content_block_stop":
		handleContentBlockStop(evt, partial, events)
	case "message_delta":
		handleMessageDelta(evt, partial, events)
	}
}

func handleMessageStart(
	evt anthropic.MessageStreamEventUnion,
	partial *message.AssistantMessage,
	events chan<- provider.AssistantMessageEvent,
) {
	partial.Model = evt.Message.Model
	partial.ResponseID = evt.Message.ID
	events <- provider.EventStart{Partial: partial}
}

func handleContentBlockStart(
	evt anthropic.MessageStreamEventUnion,
	partial *message.AssistantMessage,
	events chan<- provider.AssistantMessageEvent,
) {
	idx := int(evt.Index)
	blockType := evt.ContentBlock.Type

	switch blockType {
	case "text":
		partial.Content = append(partial.Content, message.TextContent{Type: "text"})
		events <- provider.EventTextStart{Partial: partial, ContentIndex: idx}
	case "thinking":
		partial.Content = append(partial.Content, message.ThinkingContent{Type: "thinking"})
		events <- provider.EventThinkingStart{Partial: partial, ContentIndex: idx}
	case "tool_use":
		partial.Content = append(partial.Content, message.ToolCall{
			Type: "toolCall",
			ID:   evt.ContentBlock.ID,
			Name: evt.ContentBlock.Name,
		})
		events <- provider.EventToolCallStart{Partial: partial, ContentIndex: idx}
	}
}

func handleContentBlockDelta(
	evt anthropic.MessageStreamEventUnion,
	partial *message.AssistantMessage,
	events chan<- provider.AssistantMessageEvent,
) {
	idx := int(evt.Index)
	delta := evt.Delta

	switch delta.Type {
	case "text_delta":
		events <- provider.EventTextDelta{
			Partial:      partial,
			ContentIndex: idx,
			Delta:        delta.Text,
		}
	case "thinking_delta":
		events <- provider.EventThinkingDelta{
			Partial:      partial,
			ContentIndex: idx,
			Delta:        delta.Thinking,
		}
	case "input_json_delta":
		events <- provider.EventToolCallDelta{
			Partial:      partial,
			ContentIndex: idx,
			Delta:        delta.PartialJSON,
		}
	case "signature_delta":
		if idx < len(partial.Content) {
			if tc, ok := partial.Content[idx].(message.ThinkingContent); ok {
				tc.ThinkingSignature += delta.Signature
				partial.Content[idx] = tc
			}
		}
	}
}

func handleContentBlockStop(
	evt anthropic.MessageStreamEventUnion,
	partial *message.AssistantMessage,
	events chan<- provider.AssistantMessageEvent,
) {
	idx := int(evt.Index)
	if idx >= len(partial.Content) {
		return
	}

	switch partial.Content[idx].(type) {
	case message.TextContent:
		events <- provider.EventTextEnd{Partial: partial, ContentIndex: idx}
	case message.ThinkingContent:
		events <- provider.EventThinkingEnd{Partial: partial, ContentIndex: idx}
	case message.ToolCall:
		tc := partial.Content[idx].(message.ToolCall)
		events <- provider.EventToolCallEnd{
			Partial:      partial,
			ContentIndex: idx,
			ToolCall:     tc,
		}
	}
}

func handleMessageDelta(
	evt anthropic.MessageStreamEventUnion,
	partial *message.AssistantMessage,
	events chan<- provider.AssistantMessageEvent,
) {
	reason := mapStopReason(string(evt.Delta.StopReason))
	partial.StopReason = reason

	events <- provider.EventDone{
		Reason:  reason,
		Message: *partial,
	}
}

func mapStopReason(reason string) message.StopReason {
	switch reason {
	case "end_turn", "stop_sequence":
		return message.StopReasonStop
	case "max_tokens":
		return message.StopReasonLength
	case "tool_use":
		return message.StopReasonToolUse
	default:
		return message.StopReasonStop
	}
}
