package agent

import (
	"context"
	"sync"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

type Agent struct {
	opts     AgentOptions
	state    *AgentState
	queue    *PendingMessageQueue
	streamFn provider.StreamFn

	mu     sync.Mutex
	cancel context.CancelFunc

	subsMu sync.RWMutex
	subs   []chan AgentEvent
}

func NewAgent(opts AgentOptions, streamFn provider.StreamFn) *Agent {
	return &Agent{
		opts:     opts,
		state:    NewAgentState(),
		queue:    NewPendingMessageQueue(),
		streamFn: streamFn,
		subs:     []chan AgentEvent{},
	}
}

func (a *Agent) Prompt(ctx context.Context, text string) <-chan AgentEvent {
	msg := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{message.TextContent{Text: text}},
	}
	a.state.AddMessage(msg)
	return a.startLoop(ctx)
}

func (a *Agent) Subscribe() <-chan AgentEvent {
	ch := make(chan AgentEvent, 256)
	a.subsMu.Lock()
	defer a.subsMu.Unlock()
	a.subs = append(a.subs, ch)
	return ch
}

func (a *Agent) Steer(_ context.Context, text string) {
	msg := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{message.TextContent{Text: text}},
	}
	a.queue.Enqueue(QueueModeSteering, msg)
}

func (a *Agent) FollowUp(_ context.Context, text string) {
	msg := message.UserMessage{
		Role:    "user",
		Content: []message.ContentBlock{message.TextContent{Text: text}},
	}
	a.queue.Enqueue(QueueModeFollowUp, msg)
}

func (a *Agent) Abort() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}

func (a *Agent) State() *AgentState {
	return a.state
}

func (a *Agent) SetModel(model provider.Model) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.opts.Model = model
}

func (a *Agent) ModelID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.opts.Model.ID
}

func (a *Agent) SetThinkingEnabled(v bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.opts.StreamOpts.ThinkingEnabled = v
}

func (a *Agent) ThinkingEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.opts.StreamOpts.ThinkingEnabled
}

func (a *Agent) startLoop(parent context.Context) <-chan AgentEvent {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	ctx, cancel := context.WithCancel(parent)
	a.cancel = cancel
	a.mu.Unlock()

	out := make(chan AgentEvent, 256)

	go func() {
		defer close(out)
		defer a.cleanupSubs()

		for evt := range AgentLoop(ctx, a.opts, a.state, a.streamFn, a.queue) {
			out <- evt
			a.broadcast(evt)
		}
	}()

	return out
}

func (a *Agent) broadcast(evt AgentEvent) {
	a.subsMu.RLock()
	defer a.subsMu.RUnlock()
	for _, ch := range a.subs {
		select {
		case ch <- evt:
		default:
		}
	}
}

func (a *Agent) cleanupSubs() {
	a.subsMu.Lock()
	defer a.subsMu.Unlock()
	for _, ch := range a.subs {
		close(ch)
	}
	a.subs = []chan AgentEvent{}
}
