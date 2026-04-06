package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/fantods/yaah/internal/message"
)

type toolExecEnv struct {
	tools      []AgentTool
	beforeHook BeforeToolCallHook
	afterHook  AfterToolCallHook
}

func executeToolCalls(
	ctx context.Context,
	opts AgentOptions,
	calls []message.ToolCall,
	out *AgentEventStream,
) []message.ToolResultMessage {
	env := toolExecEnv{
		tools:      opts.Tools,
		beforeHook: opts.BeforeToolCall,
		afterHook:  opts.AfterToolCall,
	}

	if opts.LoopConfig.ParallelToolCalls && len(calls) > 1 {
		return env.executeParallel(ctx, calls, out)
	}
	return env.executeSequential(ctx, calls, out)
}

func (e *toolExecEnv) executeSequential(
	ctx context.Context,
	calls []message.ToolCall,
	out *AgentEventStream,
) []message.ToolResultMessage {
	toolMap := buildToolMap(e.tools)
	results := make([]message.ToolResultMessage, 0, len(calls))

	for _, tc := range calls {
		result := e.executeOne(ctx, toolMap, tc, out)
		results = append(results, result)
	}

	return results
}

func (e *toolExecEnv) executeParallel(
	ctx context.Context,
	calls []message.ToolCall,
	out *AgentEventStream,
) []message.ToolResultMessage {
	toolMap := buildToolMap(e.tools)
	results := make([]message.ToolResultMessage, len(calls))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, tc := range calls {
		wg.Add(1)
		go func(idx int, call message.ToolCall) {
			defer wg.Done()
			result := e.executeOne(ctx, toolMap, call, out)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, tc)
	}

	wg.Wait()
	return results
}

func (e *toolExecEnv) executeOne(
	ctx context.Context,
	toolMap map[string]AgentTool,
	tc message.ToolCall,
	out *AgentEventStream,
) message.ToolResultMessage {
	call := AgentToolCall{
		ID:   tc.ID,
		Name: tc.Name,
		Args: tc.Arguments,
	}

	out.Push(ToolExecStartEvent{
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Args:       tc.Arguments,
	})

	if e.beforeHook != nil {
		shortCircuit, err := e.beforeHook(ctx, BeforeToolCallContext{
			ToolName: tc.Name,
			ToolCall: call,
			Args:     tc.Arguments,
		})
		if err != nil {
			return e.emitToolError(tc, err.Error(), out, call)
		}
		if shortCircuit != nil {
			content := shortCircuit.Content
			if content == nil {
				content = []message.ContentBlock{}
			}
			out.Push(ToolExecEndEvent{
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Result:     content,
				IsError:    shortCircuit.IsError,
			})
			return message.ToolResultMessage{
				Role:       "toolResult",
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Content:    content,
				IsError:    shortCircuit.IsError,
			}
		}
	}

	tool, found := toolMap[tc.Name]
	if !found {
		return e.emitToolError(tc, fmt.Sprintf("tool not found: %s", tc.Name), out, call)
	}

	result, err := tool.Run(ctx, call)
	if err != nil {
		return e.emitToolError(tc, err.Error(), out, call)
	}

	content := result.Content
	if content == nil {
		content = []message.ContentBlock{}
	}

	out.Push(ToolExecEndEvent{
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Result:     content,
		IsError:    result.IsError,
	})

	if e.afterHook != nil {
		e.afterHook(ctx, AfterToolCallContext{
			ToolName: tc.Name,
			ToolCall: call,
			Result:   result,
		})
	}

	return message.ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Content:    content,
		IsError:    result.IsError,
	}
}

func (e *toolExecEnv) emitToolError(
	tc message.ToolCall,
	errMsg string,
	out *AgentEventStream,
	call AgentToolCall,
) message.ToolResultMessage {
	out.Push(ToolExecEndEvent{
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Result:     errMsg,
		IsError:    true,
	})

	if e.afterHook != nil {
		e.afterHook(nil, AfterToolCallContext{
			ToolName: tc.Name,
			ToolCall: call,
			Result: &AgentToolResult{
				Content: []message.ContentBlock{message.TextContent{Text: errMsg}},
				IsError: true,
			},
		})
	}

	return message.ToolResultMessage{
		Role:       "toolResult",
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Content:    []message.ContentBlock{message.TextContent{Text: errMsg}},
		IsError:    true,
	}
}

func buildToolMap(tools []AgentTool) map[string]AgentTool {
	m := make(map[string]AgentTool, len(tools))
	for _, t := range tools {
		m[t.Info().Name] = t
	}
	return m
}
