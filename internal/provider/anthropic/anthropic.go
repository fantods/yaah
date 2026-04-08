package anthropic

import (
	"context"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/fantods/yaah/internal/logging"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func init() {
	provider.Register(provider.Provider{
		API:          "anthropic-messages",
		Stream:       Stream,
		StreamSimple: StreamSimple,
	})
}

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
	logging.Debug("anthropic: streaming model=%s maxTokens=%d numMessages=%d numTools=%d", model.ID, maxTokens, len(ctx.Messages), len(ctx.Tools))

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

	if opts != nil && opts.ThinkingEnabled {
		budget := maxTokens - 1024
		if budget < 1024 {
			budget = 1024
		}
		logging.Debug("anthropic: thinking enabled, budget=%d", budget)
		params.Thinking = anthropic.ThinkingConfigParamOfEnabled(budget)
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
		logging.Debug("anthropic: stream error: %v", err)
		partial.ErrorMessage = fmt.Sprintf("anthropic: %s", err.Error())
		partial.StopReason = message.StopReasonError
		stream.Push(provider.EventError{
			Reason:  message.StopReasonError,
			Message: *partial,
			Err:     err,
		})
	} else {
		logging.Debug("anthropic: stream completed successfully, stopReason=%s", partial.StopReason)
	}

	<-done
	stream.End(nil)
}

func newClient(opts *provider.StreamOptions) anthropic.Client {
	clientOpts := []option.RequestOption{}

	if opts != nil {
		if opts.APIKey != nil {
			clientOpts = append(clientOpts, option.WithAPIKey(*opts.APIKey))
			logging.Debug("anthropic: using explicit API key")
		} else {
			logging.Debug("anthropic: no explicit API key, relying on ANTHROPIC_API_KEY env var")
		}
		if opts.Headers != nil {
			for k, v := range opts.Headers {
				clientOpts = append(clientOpts, option.WithHeader(k, v))
			}
		}
	} else {
		logging.Debug("anthropic: no stream options provided, relying on ANTHROPIC_API_KEY env var")
	}

	return anthropic.NewClient(clientOpts...)
}
