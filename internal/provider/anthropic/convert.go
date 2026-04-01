package anthropic

import (
	"encoding/json"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func ConvertMessages(msgs []message.Message) []anthropic.MessageParam {
	params := make([]anthropic.MessageParam, 0, len(msgs))
	for _, msg := range msgs {
		switch m := msg.(type) {
		case message.UserMessage:
			params = append(params, convertUserMessage(m))
		case message.AssistantMessage:
			params = append(params, convertAssistantMessage(m))
		case message.ToolResultMessage:
			params = append(params, convertToolResultMessage(m))
		}
	}
	return params
}

func convertUserMessage(m message.UserMessage) anthropic.MessageParam {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.Content))
	for _, block := range m.Content {
		switch b := block.(type) {
		case message.TextContent:
			blocks = append(blocks, anthropic.NewTextBlock(b.Text))
		case message.ImageContent:
			blocks = append(blocks, anthropic.NewImageBlockBase64(b.MIMEType, b.Data))
		}
	}
	return anthropic.NewUserMessage(blocks...)
}

func convertAssistantMessage(m message.AssistantMessage) anthropic.MessageParam {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.Content))
	for _, block := range m.Content {
		switch b := block.(type) {
		case message.TextContent:
			blocks = append(blocks, anthropic.NewTextBlock(b.Text))
		case message.ThinkingContent:
			blocks = append(blocks, anthropic.NewThinkingBlock(b.ThinkingSignature, b.Thinking))
		case message.ToolCall:
			input, _ := json.Marshal(b.Arguments)
			var inputAny any
			json.Unmarshal(input, &inputAny)
			blocks = append(blocks, anthropic.NewToolUseBlock(b.ID, inputAny, b.Name))
		}
	}
	return anthropic.NewAssistantMessage(blocks...)
}

func convertToolResultMessage(m message.ToolResultMessage) anthropic.MessageParam {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.Content))
	for _, block := range m.Content {
		if b, ok := block.(message.TextContent); ok {
			blocks = append(blocks, anthropic.NewToolResultBlock(m.ToolCallID, b.Text, m.IsError))
		}
	}
	if len(blocks) == 0 {
		blocks = append(blocks, anthropic.NewToolResultBlock(m.ToolCallID, "", m.IsError))
	}
	return anthropic.NewUserMessage(blocks...)
}

func ConvertTools(tools []provider.Tool) []anthropic.ToolParam {
	params := make([]anthropic.ToolParam, 0, len(tools))
	for _, tool := range tools {
		params = append(params, anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.Opt(tool.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type:       "object",
				Properties: jsonToAny(tool.Parameters),
			},
		})
	}
	return params
}

func jsonToAny(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	json.Unmarshal(raw, &v)
	return v
}
