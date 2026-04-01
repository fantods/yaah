package provider

import (
	"sync"
	"testing"
	"time"

	"github.com/fantods/yaah/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStream() *EventStream[AssistantMessageEvent, message.AssistantMessage] {
	return NewEventStream[AssistantMessageEvent, message.AssistantMessage](
		func(evt AssistantMessageEvent) bool {
			_, ok := evt.(EventDone)
			return ok
		},
		func(evt AssistantMessageEvent) message.AssistantMessage {
			if done, ok := evt.(EventDone); ok {
				return done.Message
			}
			return message.AssistantMessage{}
		},
	)
}

func TestEventStreamPushAndEvents(t *testing.T) {
	s := newTestStream()

	go func() {
		s.Push(EventTextDelta{Delta: "hello "})
		s.Push(EventTextDelta{Delta: "world"})
		s.End(nil)
	}()

	var deltas []string
	for evt := range s.Events() {
		if d, ok := evt.(EventTextDelta); ok {
			deltas = append(deltas, d.Delta)
		}
	}

	assert.Equal(t, []string{"hello ", "world"}, deltas)
}

func TestEventStreamResult(t *testing.T) {
	s := newTestStream()
	expected := message.AssistantMessage{Role: "assistant", Model: "claude-3", StopReason: message.StopReasonStop}

	s.Push(EventDone{Reason: message.StopReasonStop, Message: expected})

	result, ok := <-s.Result()
	assert.True(t, ok)
	assert.Equal(t, "claude-3", result.Model)
	assert.Equal(t, message.StopReasonStop, result.StopReason)
}

func TestEventStreamResultBlocksUntilEnd(t *testing.T) {
	s := newTestStream()

	done := make(chan struct{})
	go func() {
		<-s.Result()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Result should block until End or completing event")
	case <-time.After(50 * time.Millisecond):
	}

	s.Push(EventDone{Reason: message.StopReasonStop, Message: message.AssistantMessage{}})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Result should unblock after completing event")
	}
}

func TestEventStreamEndClosesEventsChannel(t *testing.T) {
	s := newTestStream()

	go func() {
		s.Push(EventTextDelta{Delta: "hi"})
		s.End(nil)
	}()

	collected := 0
	for range s.Events() {
		collected++
	}
	assert.Equal(t, 1, collected)
}

func TestEventStreamEndWithResult(t *testing.T) {
	s := newTestStream()
	expected := message.AssistantMessage{Role: "assistant", Model: "gpt-4"}

	go func() {
		s.End(&expected)
	}()

	result := <-s.Result()
	assert.Equal(t, "gpt-4", result.Model)
}

func TestEventStreamEndIdempotent(t *testing.T) {
	s := newTestStream()

	go func() {
		s.End(nil)
		s.End(nil)
		s.End(nil)
	}()

	_, ok := <-s.Result()
	assert.False(t, ok, "Result channel should close once")
}

func TestEventStreamConcurrentPush(t *testing.T) {
	s := newTestStream()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.Push(EventTextDelta{Delta: "x"})
		}(i)
	}

	wg.Wait()
	s.End(nil)

	count := 0
	for range s.Events() {
		count++
	}
	assert.Equal(t, 100, count)
}

func TestEventStreamCompletingEventExtractsResult(t *testing.T) {
	s := newTestStream()
	expected := message.AssistantMessage{Role: "assistant", Model: "claude-3", Usage: message.Usage{Input: 42}}

	go func() {
		s.Push(EventTextDelta{Delta: "hi"})
		s.Push(EventDone{Reason: message.StopReasonStop, Message: expected})
	}()

	result := <-s.Result()
	assert.Equal(t, "claude-3", result.Model)
	assert.Equal(t, int64(42), result.Usage.Input)
}

func TestEventStreamMultipleConsumers(t *testing.T) {
	s := newTestStream()

	go func() {
		for i := 0; i < 10; i++ {
			s.Push(EventTextDelta{Delta: "a"})
		}
		s.End(nil)
	}()

	var m sync.Mutex
	counts := make([]int, 2)
	var wg sync.WaitGroup

	for c := 0; c < 2; c++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for range s.Events() {
				m.Lock()
				counts[idx]++
				m.Unlock()
			}
		}(c)
	}

	wg.Wait()

	total := counts[0] + counts[1]
	assert.Equal(t, 10, total)
}

func TestAssistantMessageEventStreamAlias(t *testing.T) {
	var _ *EventStream[AssistantMessageEvent, message.AssistantMessage] = newTestStream()
}

func TestEventStreamEmptyStream(t *testing.T) {
	s := newTestStream()

	go func() {
		s.End(nil)
	}()

	count := 0
	for range s.Events() {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestEventStreamPushAfterEnd(t *testing.T) {
	s := newTestStream()

	s.End(nil)

	require.NotPanics(t, func() {
		s.Push(EventTextDelta{Delta: "late"})
	})
}
