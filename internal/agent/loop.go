package agent

import (
	"context"
	"fmt"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

type AgentEventStream struct {
	ch        chan AgentEvent
	closeOnce chan struct{}
}

func NewAgentEventStream() *AgentEventStream {
	return &AgentEventStream{
		ch:        make(chan AgentEvent, 256),
		closeOnce: make(chan struct{}),
	}
}

func (s *AgentEventStream) Push(evt AgentEvent) {
	select {
	case <-s.closeOnce:
		return
	default:
	}
	select {
	case s.ch <- evt:
	default:
	}
}

func (s *AgentEventStream) Close() {
	select {
	case <-s.closeOnce:
		return
	default:
		close(s.closeOnce)
		close(s.ch)
	}
}

func (s *AgentEventStream) Events() <-chan AgentEvent {
	return s.ch
}

func AgentLoop(
	ctx context.Context,
	opts AgentOptions,
	state *AgentState,
	streamFn provider.StreamFn,
) <-chan AgentEvent {
	out := NewAgentEventStream()

	go func() {
		defer out.Close()
		out.Push(AgentStartEvent{})

		state.SetStreaming(true)
		defer state.SetStreaming(false)

		runLoop(ctx, opts, state, streamFn, out)

		out.Push(AgentEndEvent{Messages: state.GetMessages()})
	}()

	return out.Events()
}

func runLoop(
	ctx context.Context,
	opts AgentOptions,
	state *AgentState,
	streamFn provider.StreamFn,
	out *AgentEventStream,
) {
	maxTurns := opts.LoopConfig.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 1
	}

	for range maxTurns {
		state.IncrementTurn()
		out.Push(TurnStartEvent{})

		pctx := buildProviderContext(opts, state)
		stream := streamFn(opts.Model, pctx, &opts.StreamOpts)

		msg, err := streamAssistantResponse(ctx, stream, out)
		if err != nil {
			state.SetError(err)
			return
		}

		state.AddMessage(msg)

		if msg.StopReason != message.StopReasonToolUse {
			out.Push(TurnEndEvent{
				Message:     msg,
				ToolResults: []message.ToolResultMessage{},
			})
			return
		}

		toolCalls := extractToolCalls(msg)
		toolResults := executeToolCallsInline(ctx, opts, toolCalls, out)

		for _, tr := range toolResults {
			state.AddMessage(tr)
		}

		out.Push(TurnEndEvent{
			Message:     msg,
			ToolResults: toolResults,
		})
	}
}

func streamAssistantResponse(
	_ context.Context,
	stream *provider.AssistantMessageEventStream,
	out *AgentEventStream,
) (message.AssistantMessage, error) {
	var current message.AssistantMessage
	started := false

	for evt := range stream.Events() {
		switch e := evt.(type) {
		case provider.EventStart:
			current = *e.Partial
		case provider.EventTextStart:
			if !started {
				out.Push(MessageStartEvent{Message: current})
				started = true
			}
		case provider.EventTextDelta:
			if !started {
				out.Push(MessageStartEvent{Message: current})
				started = true
			}
			out.Push(MessageUpdateEvent{
				Message:               current,
				AssistantMessageEvent: e,
			})
		case provider.EventToolCallStart, provider.EventToolCallDelta:
			if !started {
				out.Push(MessageStartEvent{Message: current})
				started = true
			}
			out.Push(MessageUpdateEvent{
				Message:               current,
				AssistantMessageEvent: e,
			})
		case provider.EventToolCallEnd:
			if !started {
				out.Push(MessageStartEvent{Message: current})
				started = true
			}
			out.Push(MessageUpdateEvent{
				Message:               current,
				AssistantMessageEvent: e,
			})
		case provider.EventDone:
			current = e.Message
		case provider.EventError:
			if !started {
				out.Push(MessageStartEvent{Message: current})
			}
			out.Push(MessageEndEvent{Message: current})
			return current, nil
		}
	}

	select {
	case result := <-stream.Result():
		current = result
	default:
	}

	if !started {
		out.Push(MessageStartEvent{Message: current})
	}
	out.Push(MessageEndEvent{Message: current})

	return current, nil
}

func extractToolCalls(msg message.AssistantMessage) []message.ToolCall {
	var calls []message.ToolCall
	for _, block := range msg.Content {
		if tc, ok := block.(message.ToolCall); ok {
			calls = append(calls, tc)
		}
	}
	return calls
}

func executeToolCallsInline(
	ctx context.Context,
	opts AgentOptions,
	calls []message.ToolCall,
	out *AgentEventStream,
) []message.ToolResultMessage {
	toolMap := make(map[string]AgentTool, len(opts.Tools))
	for _, t := range opts.Tools {
		toolMap[t.Info().Name] = t
	}

	results := make([]message.ToolResultMessage, 0, len(calls))
	for _, tc := range calls {
		out.Push(ToolExecStartEvent{
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Args:       tc.Arguments,
		})

		tool, found := toolMap[tc.Name]
		if !found {
			errMsg := fmt.Sprintf("tool not found: %s", tc.Name)
			out.Push(ToolExecEndEvent{
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Result:     errMsg,
				IsError:    true,
			})
			results = append(results, message.ToolResultMessage{
				Role:       "toolResult",
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Content: []message.ContentBlock{
					message.TextContent{Text: errMsg},
				},
				IsError: true,
			})
			continue
		}

		result, err := tool.Run(ctx, AgentToolCall{
			ID:   tc.ID,
			Name: tc.Name,
			Args: tc.Arguments,
		})
		if err != nil {
			out.Push(ToolExecEndEvent{
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Result:     err.Error(),
				IsError:    true,
			})
			results = append(results, message.ToolResultMessage{
				Role:       "toolResult",
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Content: []message.ContentBlock{
					message.TextContent{Text: err.Error()},
				},
				IsError: true,
			})
			continue
		}

		out.Push(ToolExecEndEvent{
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Result:     result.Content,
			IsError:    result.IsError,
		})

		content := result.Content
		if content == nil {
			content = []message.ContentBlock{}
		}

		results = append(results, message.ToolResultMessage{
			Role:       "toolResult",
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Content:    content,
			IsError:    result.IsError,
		})
	}

	return results
}

func buildProviderContext(opts AgentOptions, state *AgentState) provider.Context {
	tools := make([]provider.Tool, 0, len(opts.Tools))
	for _, t := range opts.Tools {
		tools = append(tools, t.Info())
	}

	return provider.Context{
		SystemPrompt: opts.SystemPrompt,
		Messages:     state.GetMessages(),
		Tools:        tools,
	}
}
