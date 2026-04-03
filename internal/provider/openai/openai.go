package openai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

func Stream(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
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

	go runStream(model, ctx, opts, stream)

	return stream
}

func StreamSimple(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
	return Stream(model, ctx, opts)
}

func runStream(
	model provider.Model,
	pCtx provider.Context,
	opts *provider.StreamOptions,
	stream *provider.AssistantMessageEventStream,
) {
	client := newClient(model, opts)
	params := buildParams(model, pCtx, opts)

	sseStream := client.Chat.Completions.NewStreaming(context.Background(), params)
	defer sseStream.Close()

	parser := NewChunkParser(model)
	parser.SetTimestamp(time.Now().UnixMilli())

	stream.Push(provider.EventStart{
		Partial: parser.Partial(),
	})

	for sseStream.Next() {
		chunk := sseStream.Current()
		events := parser.ParseChunk(chunk)
		for _, evt := range events {
			stream.Push(evt)
		}
	}

	if err := sseStream.Err(); err != nil {
		for _, evt := range parser.FinalizeWithError(err) {
			stream.Push(evt)
		}
		stream.End(nil)
		return
	}

	for _, evt := range parser.Finalize() {
		stream.Push(evt)
	}

	stream.End(nil)
}

func buildParams(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) openai.ChatCompletionNewParams {
	compat := DetectCompat(model)

	msgParams := ConvertMessages(ctx.Messages)

	if ctx.SystemPrompt != "" {
		var sysMsg openai.ChatCompletionMessageParamUnion
		if compat.SupportsDeveloperRole && model.Reasoning {
			sysMsg = openai.DeveloperMessage(ctx.SystemPrompt)
		} else {
			sysMsg = openai.SystemMessage(ctx.SystemPrompt)
		}
		msgParams = append([]openai.ChatCompletionMessageParamUnion{sysMsg}, msgParams...)
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(model.ID),
		Messages: msgParams,
	}

	if len(ctx.Tools) > 0 {
		params.Tools = ConvertTools(ctx.Tools)
	}

	if compat.SupportsUsageInStreaming {
		params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: param.NewOpt(true),
		}
	}

	if opts != nil {
		if opts.Temperature != nil {
			params.Temperature = param.NewOpt(*opts.Temperature)
		}

		if opts.MaxTokens != nil {
			switch compat.MaxTokensField {
			case MaxTokensFieldLegacy:
				params.MaxTokens = param.NewOpt(int64(*opts.MaxTokens))
			default:
				params.MaxCompletionTokens = param.NewOpt(int64(*opts.MaxTokens))
			}
		}
	}

	if compat.SupportsStore {
		params.Store = param.NewOpt(false)
	}

	return params
}

func newClient(model provider.Model, opts *provider.StreamOptions) openai.Client {
	clientOpts := []option.RequestOption{}

	if model.BaseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(model.BaseURL))
	}

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

	if model.Headers != nil {
		for k, v := range model.Headers {
			clientOpts = append(clientOpts, option.WithHeader(k, v))
		}
	}

	return openai.NewClient(clientOpts...)
}

func formatOpenAIError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "API error") {
		return fmt.Sprintf("openai: %s", msg)
	}
	return fmt.Sprintf("openai: %s", msg)
}
