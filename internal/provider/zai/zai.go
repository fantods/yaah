package zai

import (
	"context"
	"time"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	openaiprovider "github.com/fantods/yaah/internal/provider/openai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

func init() {
	provider.Register(provider.Provider{
		API:          "zai",
		Stream:       Stream,
		StreamSimple: StreamSimple,
	})
}

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

	parser := openaiprovider.NewChunkParser(model)
	parser.Partial().API = "zai"
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
	compat := openaiprovider.DetectCompat(model)

	msgParams := ConvertMessages(ctx.Messages)

	if ctx.SystemPrompt != "" {
		sysMsg := openai.SystemMessage(ctx.SystemPrompt)
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
			case openaiprovider.MaxTokensFieldLegacy:
				params.MaxTokens = param.NewOpt(int64(*opts.MaxTokens))
			default:
				params.MaxCompletionTokens = param.NewOpt(int64(*opts.MaxTokens))
			}
		}
	}

	return params
}

func newClient(model provider.Model, opts *provider.StreamOptions) openai.Client {
	clientOpts := []option.RequestOption{}

	baseURL := model.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	clientOpts = append(clientOpts, option.WithBaseURL(baseURL))

	apiKey := ResolveAPIKey(opts)
	if apiKey != nil {
		clientOpts = append(clientOpts, option.WithAPIKey(*apiKey))
	}

	if opts != nil {
		for k, v := range opts.Headers {
			clientOpts = append(clientOpts, option.WithHeader(k, v))
		}
	}

	for k, v := range model.Headers {
		clientOpts = append(clientOpts, option.WithHeader(k, v))
	}

	return openai.NewClient(clientOpts...)
}
