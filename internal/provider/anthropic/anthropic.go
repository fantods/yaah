package anthropic

import (
	"context"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func Stream(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
	stream := provider.NewEventStream[provider.AssistantMessageEvent, message.AssistantMessage](
		func(evt provider.AssistantMessageEvent) bool {
			_, ok := evt.(provider.EventDone)
			if !ok {
				_, ok = evt.(provider.EventError)
			}
			return ok
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

	go runStream(model, ctx, opts, stream)

	return stream
}

func StreamSimple(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
	return Stream(model, ctx, opts)
}

func runStream(
	model provider.Model,
	ctx provider.Context,
	opts *provider.StreamOptions,
	stream *provider.AssistantMessageEventStream,
) {
	partial := &message.AssistantMessage{
		Role:     "assistant",
		API:      "anthropic-messages",
		Provider: "anthropic",
		Model:    model.ID,
	}

	client := newClient(opts)
	msgParams := ConvertMessages(ctx.Messages)

	var toolParams []anthropic.ToolUnionParam
	for _, t := range ConvertTools(ctx.Tools) {
		toolParams = append(toolParams, anthropic.ToolUnionParam{OfTool: &t})
	}

	maxTokens := int64(model.MaxTokens)
	if opts != nil && opts.MaxTokens != nil {
		maxTokens = int64(*opts.MaxTokens)
	}

	params := anthropic.MessageNewParams{
		MaxTokens: maxTokens,
		Messages:  msgParams,
		Tools:     toolParams,
	}

	if ctx.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: ctx.SystemPrompt},
		}
	}

	sseStream := client.Messages.NewStreaming(context.Background(), params)

	eventsCh := make(chan provider.AssistantMessageEvent, 64)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for evt := range eventsCh {
			stream.Push(evt)
		}
	}()

	for sseStream.Next() {
		evt := sseStream.Current()
		HandleEvent(evt, partial, eventsCh)
	}

	close(eventsCh)

	if err := sseStream.Err(); err != nil {
		partial.ErrorMessage = formatAnthropicError(err)
		partial.StopReason = message.StopReasonError
		stream.Push(provider.EventError{
			Reason:  message.StopReasonError,
			Message: *partial,
		})
	}

	<-done
	stream.End(nil)
}

func newClient(opts *provider.StreamOptions) anthropic.Client {
	clientOpts := []option.RequestOption{}

	if opts != nil {
		if opts.APIKey != nil {
			clientOpts = append(clientOpts, option.WithAPIKey(*opts.APIKey))
		}
		if opts.Headers != nil {
			for k, v := range opts.Headers {
				clientOpts = append(clientOpts, option.WithHeader(k, v))
			}
		}
	}

	return anthropic.NewClient(clientOpts...)
}

func formatAnthropicError(err error) string {
	return fmt.Sprintf("anthropic: %s", err.Error())
}
