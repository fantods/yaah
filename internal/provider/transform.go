package provider

import (
	"github.com/fantods/yaah/internal/message"
)

func NormalizeToolCallID(id string) string {
	if id == "" {
		return ""
	}
	if len(id) >= 5 && (id[:5] == "call_" || id[:6] == "toolu_") {
		return id
	}
	return "call_" + id
}

func ExtractThinkingBlocks(msg message.AssistantMessage) ([]message.ThinkingContent, []message.ContentBlock) {
	var thinking []message.ThinkingContent
	var other []message.ContentBlock

	for _, block := range msg.Content {
		if tc, ok := block.(message.ThinkingContent); ok {
			thinking = append(thinking, tc)
		} else {
			other = append(other, block)
		}
	}

	if thinking == nil {
		thinking = []message.ThinkingContent{}
	}
	if other == nil {
		other = []message.ContentBlock{}
	}

	return thinking, other
}

func MergeThinkingBlocks(thinking []message.ThinkingContent, other []message.ContentBlock) []message.ContentBlock {
	result := make([]message.ContentBlock, 0, len(thinking)+len(other))
	for _, t := range thinking {
		result = append(result, t)
	}
	for _, o := range other {
		result = append(result, o)
	}
	return result
}

func StripThinkingBlocks(msgs []message.Message) []message.Message {
	result := make([]message.Message, len(msgs))
	for i, msg := range msgs {
		asm, ok := msg.(message.AssistantMessage)
		if !ok {
			result[i] = msg
			continue
		}
		_, remaining := ExtractThinkingBlocks(asm)
		asm.Content = remaining
		result[i] = asm
	}
	return result
}

func NormalizeMessages(msgs []message.Message) []message.Message {
	result := make([]message.Message, len(msgs))
	idMap := make(map[string]string)

	for i, msg := range msgs {
		switch m := msg.(type) {
		case message.AssistantMessage:
			asm := m
			asm.Content = normalizeContentBlockIDs(asm.Content, idMap)
			result[i] = asm
		case message.ToolResultMessage:
			trm := m
			if mapped, ok := idMap[trm.ToolCallID]; ok {
				trm.ToolCallID = mapped
			} else {
				normalized := NormalizeToolCallID(trm.ToolCallID)
				idMap[trm.ToolCallID] = normalized
				trm.ToolCallID = normalized
			}
			result[i] = trm
		default:
			result[i] = msg
		}
	}
	return result
}

func normalizeContentBlockIDs(blocks []message.ContentBlock, idMap map[string]string) []message.ContentBlock {
	result := make([]message.ContentBlock, len(blocks))
	for i, block := range blocks {
		tc, ok := block.(message.ToolCall)
		if !ok {
			result[i] = block
			continue
		}
		normalized := NormalizeToolCallID(tc.ID)
		idMap[tc.ID] = normalized
		tc.ID = normalized
		result[i] = tc
	}
	return result
}
