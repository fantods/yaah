package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/fantods/yaah/internal/logging"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

type AgentEventStream struct {
	ch        chan AgentEvent
	closeOnce chan struct{}
	mu        sync.Mutex
}

func NewAgentEventStream() *AgentEventStream {
	return &AgentEventStream{
		ch:        make(chan AgentEvent, 256),
		closeOnce: make(chan struct{}),
	}
}

func (s *AgentEventStream) Push(evt AgentEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
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
	queue *PendingMessageQueue,
) <-chan AgentEvent {
	out := NewAgentEventStream()

	go func() {
		defer out.Close()
		out.Push(AgentStartEvent{})
		logging.Debug("agent loop started")

		state.SetStreaming(true)
		defer state.SetStreaming(false)

		loopErr := runLoop(ctx, opts, state, streamFn, out, queue)
		if loopErr != nil {
			logging.Debug("agent loop error: %v", loopErr)
		} else {
			logging.Debug("agent loop completed")
		}

		out.Push(AgentEndEvent{
			Messages: state.GetMessages(),
			Error:    loopErr,
		})
	}()

	return out.Events()
}

func runLoop(
	ctx context.Context,
	opts AgentOptions,
	state *AgentState,
	streamFn provider.StreamFn,
	out *AgentEventStream,
	queue *PendingMessageQueue,
) error {
	maxTurns := opts.LoopConfig.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 1
	}

	turnsRemaining := maxTurns

	for turnsRemaining > 0 {
		turnsRemaining--
		state.IncrementTurn()
		out.Push(TurnStartEvent{})

		pctx := buildProviderContext(opts, state)
		stream := streamFn(opts.Model, pctx, &opts.StreamOpts)
		logging.Debug("turn %d: streaming with model=%s api=%s maxTokens=%d", state.GetTurn(), opts.Model.ID, opts.Model.API, opts.Model.MaxTokens)

		msg, err := streamAssistantResponse(ctx, stream, out)
		if err != nil {
			state.SetError(err)
			logging.Debug("turn %d: stream error: %v", state.GetTurn(), err)
			return err
		}

		logging.Debug("turn %d: stream completed, stopReason=%s", state.GetTurn(), msg.StopReason)

		state.AddMessage(msg)

		if msg.StopReason != message.StopReasonToolUse {
			out.Push(TurnEndEvent{
				Message:     msg,
				ToolResults: []message.ToolResultMessage{},
			})

			if queue != nil {
				if steerMsg, ok := queue.DequeueByMode(QueueModeSteering); ok {
					state.AddMessage(steerMsg)
					turnsRemaining = maxTurns
					continue
				}
				if followUpMsg, ok := queue.DequeueByMode(QueueModeFollowUp); ok {
					state.AddMessage(followUpMsg)
					turnsRemaining = maxTurns
					continue
				}
			}
			return nil
		}

		toolCalls := extractToolCalls(msg)
		toolResults := executeToolCalls(ctx, opts, toolCalls, out)
		logging.Debug("turn %d: executed %d tool calls", state.GetTurn(), len(toolCalls))

		for _, tr := range toolResults {
			state.AddMessage(tr)
		}

		out.Push(TurnEndEvent{
			Message:     msg,
			ToolResults: toolResults,
		})

		if queue != nil {
			if steerMsg, ok := queue.DequeueByMode(QueueModeSteering); ok {
				state.AddMessage(steerMsg)
				turnsRemaining = maxTurns
			}
		}
	}
	return nil
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
			errMsg := "stream error: provider returned EventError"
			if e.Err != nil {
				errMsg = e.Err.Error()
			} else if e.Message.ErrorMessage != "" {
				errMsg = e.Message.ErrorMessage
			}
			return current, fmt.Errorf("%s", errMsg)
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
