package zai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fantods/yaah/internal/logging"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	openaiprovider "github.com/fantods/yaah/internal/provider/openai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
	"github.com/tidwall/gjson"
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

	thinkingEnabled := opts != nil && opts.ThinkingEnabled
	if thinkingEnabled {
		go runRawStream(model, ctx, opts, stream)
	} else {
		go runStream(model, ctx, opts, stream)
	}

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

func runRawStream(
	model provider.Model,
	pCtx provider.Context,
	opts *provider.StreamOptions,
	stream *provider.AssistantMessageEventStream,
) {
	params := buildParams(model, pCtx, opts)
	body, err := json.Marshal(params)
	if err != nil {
		stream.Push(provider.EventError{Err: fmt.Errorf("zai: marshaling params: %w", err)})
		stream.End(nil)
		return
	}

	var rawParams map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawParams); err != nil {
		stream.Push(provider.EventError{Err: fmt.Errorf("zai: preparing params: %w", err)})
		stream.End(nil)
		return
	}
	rawParams["thinking"] = json.RawMessage(`{"type":"enabled"}`)
	rawParams["stream"] = json.RawMessage(`true`)
	body, err = json.Marshal(rawParams)
	if err != nil {
		stream.Push(provider.EventError{Err: fmt.Errorf("zai: marshaling params: %w", err)})
		stream.End(nil)
		return
	}

	baseURL := model.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		stream.Push(provider.EventError{Err: fmt.Errorf("zai: creating request: %w", err)})
		stream.End(nil)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	apiKey := ResolveAPIKey(opts)
	if apiKey != nil {
		token, err := generateToken(*apiKey)
		if err != nil {
			logging.Debug("zai: failed to generate token: %v", err)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	if opts != nil {
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}
	}
	for k, v := range model.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		stream.Push(provider.EventError{Err: fmt.Errorf("zai: request failed: %w", err)})
		stream.End(nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		stream.Push(provider.EventError{Err: fmt.Errorf("zai: HTTP %d: %s", resp.StatusCode, string(respBody))})
		stream.End(nil)
		return
	}

	parser := openaiprovider.NewChunkParser(model)
	parser.Partial().API = "zai"
	parser.SetTimestamp(time.Now().UnixMilli())

	stream.Push(provider.EventStart{
		Partial: parser.Partial(),
	})

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(nil, bufio.MaxScanTokenSize<<9)

	var dataBuf bytes.Buffer

	for scanner.Scan() {
		line := scanner.Bytes()

		if len(line) == 0 {
			if dataBuf.Len() > 0 {
				rawData := dataBuf.Bytes()
				dataBuf.Reset()

				if bytes.HasPrefix(rawData, []byte("[DONE]")) {
					continue
				}

				events := parseRawChunk(parser, rawData)
				for _, evt := range events {
					stream.Push(evt)
				}
			}
			continue
		}

		after, ok := bytes.CutPrefix(line, []byte("data: "))
		if ok {
			dataBuf.Write(after)
			dataBuf.WriteByte('\n')
		}
	}

	if err := scanner.Err(); err != nil {
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

func parseRawChunk(parser *openaiprovider.ChunkParser, rawData []byte) []provider.AssistantMessageEvent {
	reasoningContent := gjson.GetBytes(rawData, "choices.0.delta.reasoning_content")
	if reasoningContent.Exists() && reasoningContent.String() != "" {
		return parser.ParseReasoningDelta(reasoningContent.String())
	}

	textContent := gjson.GetBytes(rawData, "choices.0.delta.content")
	if textContent.Exists() && textContent.String() != "" {
		return parser.ParseReasoningTextDelta(textContent.String())
	}

	toolCallsResult := gjson.GetBytes(rawData, "choices.0.delta.tool_calls")
	if toolCallsResult.Exists() && toolCallsResult.IsArray() {
		var events []provider.AssistantMessageEvent
		toolCallsResult.ForEach(func(_, tc gjson.Result) bool {
			events = append(events, parser.ParseToolCallDelta(
				tc.Get("index").Int(),
				tc.Get("id").String(),
				tc.Get("function.name").String(),
				tc.Get("function.arguments").String(),
			)...)
			return true
		})
		if len(events) > 0 {
			return events
		}
	}

	finishReason := gjson.GetBytes(rawData, "choices.0.finish_reason")
	if finishReason.Exists() && finishReason.String() != "" {
		stopReason, errMsg := openaiprovider.MapStopReason(finishReason.String())
		if errMsg != "" {
			parser.SetStopReason(stopReason, errMsg)
		} else {
			parser.SetStopReason(stopReason, "")
		}
	}

	usage := gjson.GetBytes(rawData, "usage")
	if usage.Exists() {
		parser.ParseRawUsage(usage.Raw)
	}

	return nil
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
		token, err := generateToken(*apiKey)
		if err != nil {
			logging.Debug("zai: failed to generate token: %v", err)
		} else {
			clientOpts = append(clientOpts, option.WithHeader("Authorization", "Bearer "+token))
		}
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
