package openai

import (
	"encoding/json"
	"strings"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

// ConvertMessages transforms internal messages into OpenAI ChatCompletionMessageParamUnion.
func ConvertMessages(msgs []message.Message) []openai.ChatCompletionMessageParamUnion {
	params := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
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

// ConvertTools transforms internal tools into OpenAI ChatCompletionToolParam types.
func ConvertTools(tools []provider.Tool) []openai.ChatCompletionToolParam {
	params := make([]openai.ChatCompletionToolParam, 0, len(tools))
	for _, tool := range tools {
		params = append(params, openai.ChatCompletionToolParam{
			// Type is elided - zero value marshals as "function"
			Function: shared.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: param.NewOpt(tool.Description),
				Parameters:  jsonToFunctionParameters(tool.Parameters),
			},
		})
	}
	return params
}

func convertUserMessage(m message.UserMessage) openai.ChatCompletionMessageParamUnion {
	// Single text content: use simple string form
	if len(m.Content) == 1 {
		if text, ok := m.Content[0].(message.TextContent); ok {
			return openai.UserMessage(text.Text)
		}
	}

	// Multiple blocks or mixed content: use content parts
	parts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(m.Content))
	for _, block := range m.Content {
		switch b := block.(type) {
		case message.TextContent:
			parts = append(parts, openai.TextContentPart(b.Text))
		case message.ImageContent:
			parts = append(parts, openai.ImageContentPart(
				openai.ChatCompletionContentPartImageImageURLParam{
					URL: "data:" + b.MIMEType + ";base64," + b.Data,
				},
			))
		}
	}

	// Handle empty content
	if len(parts) == 0 {
		return openai.UserMessage("")
	}

	return openai.UserMessage(parts)
}

func convertAssistantMessage(m message.AssistantMessage) openai.ChatCompletionMessageParamUnion {
	// Extract text and tool calls from content blocks
	var toolCalls []openai.ChatCompletionMessageToolCallParam
	var textParts []string

	for _, block := range m.Content {
		switch b := block.(type) {
		case message.TextContent:
			// Filter out empty/whitespace-only text blocks
			if strings.TrimSpace(b.Text) != "" {
				textParts = append(textParts, b.Text)
			}
		case message.ToolCall:
			argsJSON, _ := json.Marshal(b.Arguments)
			toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
				ID: b.ID,
				// Type is elided - zero value marshals as "function"
				Function: openai.ChatCompletionMessageToolCallFunctionParam{
					Name:      b.Name,
					Arguments: string(argsJSON),
				},
			})
		case message.ThinkingContent:
			// Silently ignore - OpenAI Chat Completions API doesn't accept
			// thinking blocks in input messages. Reasoning models handle
			// thinking internally.
		}
	}

	// Build content - OpenAI expects string, not array of text blocks
	var content string
	if len(textParts) > 0 {
		content = strings.Join(textParts, "")
	}

	// Build the assistant message param directly
	var assistant openai.ChatCompletionAssistantMessageParam
	assistant.Content.OfString = param.NewOpt(content)
	if len(toolCalls) > 0 {
		assistant.ToolCalls = toolCalls
	}

	return openai.ChatCompletionMessageParamUnion{
		OfAssistant: &assistant,
	}
}

func convertToolResultMessage(m message.ToolResultMessage) openai.ChatCompletionMessageParamUnion {
	content := message.ExtractText(m.Content)

	if content == "" {
		content = "(no output)"
	}

	return openai.ToolMessage(content, m.ToolCallID)
}

func jsonToFunctionParameters(raw json.RawMessage) shared.FunctionParameters {
	if len(raw) == 0 {
		return nil
	}
	var v shared.FunctionParameters
	json.Unmarshal(raw, &v)
	return v
}
