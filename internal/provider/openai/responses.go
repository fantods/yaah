package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

func StreamResponses(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
	stream := provider.NewEventStream[provider.AssistantMessageEvent, message.AssistantMessage](
		func(evt provider.AssistantMessageEvent) bool {
			switch evt.(type) {
			case provider.EventDone:
				return true
			case provider.EventError:
				return true
			default:
				return false
			}
		},
		func(evt provider.AssistantMessageEvent) message.AssistantMessage {
			switch e := evt.(type) {
			case provider.EventDone:
				return e.Message
			case provider.EventError:
				return e.Message
			default:
				return message.AssistantMessage{}
			}
		},
	)

	go runResponsesStream(model, ctx, opts, stream)

	return stream
}

func StreamSimpleResponses(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
	return StreamResponses(model, ctx, opts)
}

func runResponsesStream(
	model provider.Model,
	pCtx provider.Context,
	opts *provider.StreamOptions,
	stream *provider.AssistantMessageEventStream,
) {
	client := newClient(model, opts)
	params := buildResponsesParams(model, pCtx, opts)

	sseStream := client.Responses.NewStreaming(context.Background(), params)
	defer sseStream.Close()

	partial := &message.AssistantMessage{
		Role:     "assistant",
		API:      "openai-responses",
		Provider: model.Provider,
		Model:    model.ID,
		Content:  []message.ContentBlock{},
		Usage: message.Usage{
			Cost: message.Cost{},
		},
		StopReason: message.StopReasonStop,
		Timestamp:  time.Now().UnixMilli(),
	}

	stream.Push(provider.EventStart{
		Partial: partial,
	})

	processResponsesStream(sseStream, partial, stream)

	stream.End(nil)
}

func buildResponsesParams(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) responses.ResponseNewParams {
	inputItems := convertResponsesInputMessages(ctx)

	params := responses.ResponseNewParams{
		Model: shared.ResponsesModel(model.ID),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
		Store: param.NewOpt(false),
	}

	if ctx.SystemPrompt != "" {
		params.Instructions = param.NewOpt(ctx.SystemPrompt)
	}

	if len(ctx.Tools) > 0 {
		params.Tools = convertResponsesTools(ctx.Tools)
	}

	if opts != nil {
		if opts.Temperature != nil {
			params.Temperature = param.NewOpt(*opts.Temperature)
		}

		if opts.MaxTokens != nil {
			params.MaxOutputTokens = param.NewOpt(int64(*opts.MaxTokens))
		}

		if opts.SessionID != "" {
			params.PromptCacheKey = param.NewOpt(opts.SessionID)
		}
	}

	if model.Reasoning {
		params.Reasoning = shared.ReasoningParam{
			Effort: shared.ReasoningEffortMedium,
		}
		params.Include = []responses.ResponseIncludable{
			responses.ResponseIncludableReasoningEncryptedContent,
		}
	}

	return params
}

func convertResponsesInputMessages(ctx provider.Context) responses.ResponseInputParam {
	var items responses.ResponseInputParam

	for _, msg := range ctx.Messages {
		switch m := msg.(type) {
		case message.UserMessage:
			items = append(items, convertUserMessageForResponses(m)...)
		case message.AssistantMessage:
			items = append(items, convertAssistantMessageForResponses(m)...)
		case message.ToolResultMessage:
			items = append(items, convertToolResultMessageForResponses(m)...)
		}
	}

	if len(items) == 0 {
		return responses.ResponseInputParam{}
	}

	return items
}

func convertUserMessageForResponses(m message.UserMessage) []responses.ResponseInputItemUnionParam {
	if len(m.Content) == 0 {
		return nil
	}

	if len(m.Content) == 1 {
		if text, ok := m.Content[0].(message.TextContent); ok {
			return []responses.ResponseInputItemUnionParam{
				{
					OfMessage: &responses.EasyInputMessageParam{
						Role: "user",
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: param.NewOpt(text.Text),
						},
					},
				},
			}
		}
	}

	var contentParts responses.ResponseInputMessageContentListParam
	for _, block := range m.Content {
		switch b := block.(type) {
		case message.TextContent:
			contentParts = append(contentParts, responses.ResponseInputContentUnionParam{
				OfInputText: &responses.ResponseInputTextParam{
					Text: b.Text,
				},
			})
		case message.ImageContent:
			contentParts = append(contentParts, responses.ResponseInputContentUnionParam{
				OfInputImage: &responses.ResponseInputImageParam{
					ImageURL: param.NewOpt("data:" + b.MIMEType + ";base64," + b.Data),
					Detail:   "auto",
				},
			})
		}
	}

	if len(contentParts) == 0 {
		return nil
	}

	return []responses.ResponseInputItemUnionParam{
		{
			OfInputMessage: &responses.ResponseInputItemMessageParam{
				Role:    "user",
				Content: contentParts,
			},
		},
	}
}

func convertAssistantMessageForResponses(m message.AssistantMessage) []responses.ResponseInputItemUnionParam {
	var items []responses.ResponseInputItemUnionParam

	for _, block := range m.Content {
		switch b := block.(type) {
		case message.TextContent:
			if strings.TrimSpace(b.Text) == "" {
				continue
			}
			items = append(items, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: "assistant",
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: param.NewOpt(b.Text),
					},
				},
			})
		case message.ToolCall:
			callID := b.ID
			var itemID param.Opt[string]
			if idx := strings.Index(callID, "|"); idx >= 0 {
				itemID = param.NewOpt(callID[idx+1:])
				callID = callID[:idx]
			}

			argsJSON, _ := json.Marshal(b.Arguments)
			items = append(items, responses.ResponseInputItemUnionParam{
				OfFunctionCall: &responses.ResponseFunctionToolCallParam{
					CallID:    callID,
					Name:      b.Name,
					Arguments: string(argsJSON),
					ID:        itemID,
				},
			})
		case message.ThinkingContent:
			if b.ThinkingSignature != "" {
				var reasoningParam responses.ResponseReasoningItemParam
				if err := json.Unmarshal([]byte(b.ThinkingSignature), &reasoningParam); err == nil {
					items = append(items, responses.ResponseInputItemUnionParam{
						OfReasoning: &reasoningParam,
					})
				}
			}
		}
	}

	return items
}

func convertToolResultMessageForResponses(m message.ToolResultMessage) []responses.ResponseInputItemUnionParam {
	textContent := message.ExtractText(m.Content)

	if textContent == "" {
		textContent = "(no output)"
	}

	callID := m.ToolCallID
	if idx := strings.Index(callID, "|"); idx >= 0 {
		callID = callID[:idx]
	}

	return []responses.ResponseInputItemUnionParam{
		{
			OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
				CallID: callID,
				Output: textContent,
			},
		},
	}
}

func convertResponsesTools(tools []provider.Tool) []responses.ToolUnionParam {
	params := make([]responses.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		var parameters map[string]any
		if len(tool.Parameters) > 0 {
			json.Unmarshal(tool.Parameters, &parameters)
		}

		params = append(params, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        tool.Name,
				Description: param.NewOpt(tool.Description),
				Parameters:  parameters,
				Strict:      param.NewOpt(false),
			},
		})
	}
	return params
}

type responsesStreamParser struct {
	partial      *message.AssistantMessage
	currentBlock contentBlockState
	contentIndex int
	partialJSON  string
}

type contentBlockState struct {
	blockType string
}

func processResponsesStream(
	sseStream *ssestream.Stream[responses.ResponseStreamEventUnion],
	partial *message.AssistantMessage,
	stream *provider.AssistantMessageEventStream,
) {
	parser := &responsesStreamParser{
		partial: partial,
	}

	for sseStream.Next() {
		event := sseStream.Current()
		parser.processEvent(event, stream)
	}

	if err := sseStream.Err(); err != nil {
		partial.StopReason = message.StopReasonError
		partial.ErrorMessage = fmt.Sprintf("openai-responses: %s", err.Error())
		stream.Push(provider.EventError{
			Reason:  message.StopReasonError,
			Message: *partial,
		})
		return
	}

	parser.finishCurrentBlock(stream)

	stream.Push(provider.EventDone{
		Reason:  partial.StopReason,
		Message: *partial,
	})
}

func (p *responsesStreamParser) processEvent(
	event responses.ResponseStreamEventUnion,
	stream *provider.AssistantMessageEventStream,
) {
	switch variant := event.AsAny().(type) {
	case responses.ResponseCreatedEvent:
		if variant.Response.ID != "" {
			p.partial.ResponseID = variant.Response.ID
		}

	case responses.ResponseOutputItemAddedEvent:
		p.handleOutputItemAdded(variant.Item, stream)

	case responses.ResponseTextDeltaEvent:
		p.handleTextDelta(variant.Delta, stream)

	case responses.ResponseRefusalDeltaEvent:
		p.handleRefusalDelta(variant.Delta, stream)

	case responses.ResponseReasoningSummaryTextDeltaEvent:
		p.handleReasoningDelta(variant.Delta, stream)

	case responses.ResponseReasoningSummaryDeltaEvent:
		delta, _ := variant.Delta.(string)
		p.handleReasoningDelta(delta, stream)

	case responses.ResponseFunctionCallArgumentsDeltaEvent:
		p.handleFunctionCallArgsDelta(variant.Delta, stream)

	case responses.ResponseFunctionCallArgumentsDoneEvent:
		p.handleFunctionCallArgsDone(variant.Arguments)

	case responses.ResponseOutputItemDoneEvent:
		p.handleOutputItemDone(variant.Item, stream)

	case responses.ResponseCompletedEvent:
		p.handleCompleted(variant.Response)

	case responses.ResponseFailedEvent:
		resp := variant.Response
		errMsg := "unknown error"
		if resp.Error.Code != "" || resp.Error.Message != "" {
			errMsg = fmt.Sprintf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		p.partial.StopReason = message.StopReasonError
		p.partial.ErrorMessage = errMsg

	case responses.ResponseIncompleteEvent:
		p.partial.StopReason = message.StopReasonLength

	case responses.ResponseErrorEvent:
		errMsg := fmt.Sprintf("Error Code %s: %s", variant.Code, variant.Message)
		p.partial.StopReason = message.StopReasonError
		p.partial.ErrorMessage = errMsg
	}
}

func (p *responsesStreamParser) handleOutputItemAdded(
	item responses.ResponseOutputItemUnion,
	stream *provider.AssistantMessageEventStream,
) {
	switch item.Type {
	case "reasoning":
		p.finishCurrentBlock(stream)
		p.currentBlock = contentBlockState{blockType: "thinking"}
		p.partial.Content = append(p.partial.Content, message.ThinkingContent{
			Type:     "thinking",
			Thinking: "",
		})
		p.contentIndex = len(p.partial.Content) - 1
		stream.Push(provider.EventThinkingStart{
			Partial:      p.partial,
			ContentIndex: p.contentIndex,
		})

	case "message":
		p.finishCurrentBlock(stream)
		p.currentBlock = contentBlockState{blockType: "text"}
		p.partial.Content = append(p.partial.Content, message.TextContent{
			Type: "text",
		})
		p.contentIndex = len(p.partial.Content) - 1
		stream.Push(provider.EventTextStart{
			Partial:      p.partial,
			ContentIndex: p.contentIndex,
		})

	case "function_call":
		p.finishCurrentBlock(stream)
		p.currentBlock = contentBlockState{blockType: "toolCall"}
		p.partialJSON = item.Arguments

		toolID := item.CallID
		if item.ID != "" {
			toolID = item.CallID + "|" + item.ID
		}

		p.partial.Content = append(p.partial.Content, message.ToolCall{
			Type:      "toolCall",
			ID:        toolID,
			Name:      item.Name,
			Arguments: map[string]any{},
		})
		p.contentIndex = len(p.partial.Content) - 1
		stream.Push(provider.EventToolCallStart{
			Partial:      p.partial,
			ContentIndex: p.contentIndex,
		})
	}
}

func (p *responsesStreamParser) handleTextDelta(
	delta string,
	stream *provider.AssistantMessageEventStream,
) {
	if p.currentBlock.blockType != "text" || delta == "" {
		return
	}

	if tc, ok := p.partial.Content[p.contentIndex].(message.TextContent); ok {
		tc.Text += delta
		p.partial.Content[p.contentIndex] = tc
	}

	stream.Push(provider.EventTextDelta{
		Partial:      p.partial,
		ContentIndex: p.contentIndex,
		Delta:        delta,
	})
}

func (p *responsesStreamParser) handleRefusalDelta(
	delta string,
	stream *provider.AssistantMessageEventStream,
) {
	if p.currentBlock.blockType != "text" || delta == "" {
		return
	}

	if tc, ok := p.partial.Content[p.contentIndex].(message.TextContent); ok {
		tc.Text += delta
		p.partial.Content[p.contentIndex] = tc
	}

	stream.Push(provider.EventTextDelta{
		Partial:      p.partial,
		ContentIndex: p.contentIndex,
		Delta:        delta,
	})
}

func (p *responsesStreamParser) handleReasoningDelta(
	delta string,
	stream *provider.AssistantMessageEventStream,
) {
	if p.currentBlock.blockType != "thinking" || delta == "" {
		return
	}

	if tc, ok := p.partial.Content[p.contentIndex].(message.ThinkingContent); ok {
		tc.Thinking += delta
		p.partial.Content[p.contentIndex] = tc
	}

	stream.Push(provider.EventThinkingDelta{
		Partial:      p.partial,
		ContentIndex: p.contentIndex,
		Delta:        delta,
	})
}

func (p *responsesStreamParser) handleFunctionCallArgsDelta(
	delta string,
	stream *provider.AssistantMessageEventStream,
) {
	if p.currentBlock.blockType != "toolCall" {
		return
	}

	p.partialJSON += delta

	parsed := parseStreamingJSON(p.partialJSON)
	if parsed != nil {
		if tc, ok := p.partial.Content[p.contentIndex].(message.ToolCall); ok {
			tc.Arguments = parsed
			p.partial.Content[p.contentIndex] = tc
		}
	}

	stream.Push(provider.EventToolCallDelta{
		Partial:      p.partial,
		ContentIndex: p.contentIndex,
		Delta:        delta,
	})
}

func (p *responsesStreamParser) handleFunctionCallArgsDone(arguments string) {
	if p.currentBlock.blockType != "toolCall" {
		return
	}

	if arguments != "" {
		p.partialJSON = arguments
	}

	parsed := parseStreamingJSON(p.partialJSON)
	if parsed != nil {
		if tc, ok := p.partial.Content[p.contentIndex].(message.ToolCall); ok {
			tc.Arguments = parsed
			p.partial.Content[p.contentIndex] = tc
		}
	}
}

func (p *responsesStreamParser) handleOutputItemDone(
	item responses.ResponseOutputItemUnion,
	stream *provider.AssistantMessageEventStream,
) {
	switch item.Type {
	case "reasoning":
		if p.currentBlock.blockType == "thinking" {
			summary := item.Summary
			combined := ""
			for i, s := range summary {
				if i > 0 {
					combined += "\n\n"
				}
				combined += s.Text
			}

			sigBytes, _ := json.Marshal(item)
			if tc, ok := p.partial.Content[p.contentIndex].(message.ThinkingContent); ok {
				if combined != "" {
					tc.Thinking = combined
				}
				tc.ThinkingSignature = string(sigBytes)
				p.partial.Content[p.contentIndex] = tc
			}

			content := combined
			if content == "" {
				if tc, ok := p.partial.Content[p.contentIndex].(message.ThinkingContent); ok {
					content = tc.Thinking
				}
			}

			stream.Push(provider.EventThinkingEnd{
				Partial:      p.partial,
				ContentIndex: p.contentIndex,
				Content:      content,
			})
			p.currentBlock = contentBlockState{}
		}

	case "message":
		if p.currentBlock.blockType == "text" {
			fullText := ""
			for _, c := range item.Content {
				if c.Type == "output_text" {
					fullText += c.Text
				} else if c.Type == "refusal" {
					fullText += c.Refusal
				}
			}

			if tc, ok := p.partial.Content[p.contentIndex].(message.TextContent); ok {
				tc.Text = fullText
				p.partial.Content[p.contentIndex] = tc
			}

			stream.Push(provider.EventTextEnd{
				Partial:      p.partial,
				ContentIndex: p.contentIndex,
				Content:      fullText,
			})
			p.currentBlock = contentBlockState{}
		}

	case "function_call":
		if p.currentBlock.blockType == "toolCall" {
			finalArgs := item.Arguments
			if finalArgs != "" {
				p.partialJSON = finalArgs
			}

			parsed := parseStreamingJSON(p.partialJSON)
			if parsed == nil {
				parsed = map[string]any{}
			}

			toolID := item.CallID
			if item.ID != "" {
				toolID = item.CallID + "|" + item.ID
			}

			toolCall := message.ToolCall{
				Type:      "toolCall",
				ID:        toolID,
				Name:      item.Name,
				Arguments: parsed,
			}
			p.partial.Content[p.contentIndex] = toolCall

			stream.Push(provider.EventToolCallEnd{
				Partial:      p.partial,
				ContentIndex: p.contentIndex,
				ToolCall:     toolCall,
			})
			p.currentBlock = contentBlockState{}
		}
	}
}

func (p *responsesStreamParser) handleCompleted(resp responses.Response) {
	if resp.ID != "" {
		p.partial.ResponseID = resp.ID
	}

	usage := resp.Usage
	cachedTokens := usage.InputTokensDetails.CachedTokens
	p.partial.Usage = message.Usage{
		Input:       usage.InputTokens - cachedTokens,
		Output:      usage.OutputTokens,
		CacheRead:   cachedTokens,
		CacheWrite:  0,
		TotalTokens: usage.TotalTokens,
		Cost:        message.Cost{},
	}

	p.partial.StopReason = mapResponseStatus(string(resp.Status))

	hasToolCalls := false
	for _, block := range p.partial.Content {
		if _, ok := block.(message.ToolCall); ok {
			hasToolCalls = true
			break
		}
	}
	if hasToolCalls && p.partial.StopReason == message.StopReasonStop {
		p.partial.StopReason = message.StopReasonToolUse
	}
}

func (p *responsesStreamParser) finishCurrentBlock(stream *provider.AssistantMessageEventStream) {
	if p.currentBlock.blockType == "" {
		return
	}

	switch p.currentBlock.blockType {
	case "text":
		if tc, ok := p.partial.Content[p.contentIndex].(message.TextContent); ok {
			stream.Push(provider.EventTextEnd{
				Partial:      p.partial,
				ContentIndex: p.contentIndex,
				Content:      tc.Text,
			})
		}
	case "thinking":
		if tc, ok := p.partial.Content[p.contentIndex].(message.ThinkingContent); ok {
			stream.Push(provider.EventThinkingEnd{
				Partial:      p.partial,
				ContentIndex: p.contentIndex,
				Content:      tc.Thinking,
			})
		}
	case "toolCall":
		if tc, ok := p.partial.Content[p.contentIndex].(message.ToolCall); ok {
			stream.Push(provider.EventToolCallEnd{
				Partial:      p.partial,
				ContentIndex: p.contentIndex,
				ToolCall:     tc,
			})
		}
	}

	p.currentBlock = contentBlockState{}
}

func mapResponseStatus(status string) message.StopReason {
	switch status {
	case "completed":
		return message.StopReasonStop
	case "incomplete":
		return message.StopReasonLength
	case "failed", "cancelled":
		return message.StopReasonError
	case "in_progress", "queued":
		return message.StopReasonStop
	default:
		return message.StopReasonError
	}
}

func parseStreamingJSON(s string) map[string]any {
	if s == "" {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}
