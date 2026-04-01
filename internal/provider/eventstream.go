package provider

import (
	"sync"

	"github.com/fantods/yaah/internal/message"
)

type EventStream[T any, R any] struct {
	ch            chan T
	resultCh      chan R
	isComplete    func(T) bool
	extractResult func(T) R
	closeOnce     sync.Once
	closed        chan struct{}
}

func NewEventStream[T any, R any](
	isComplete func(T) bool,
	extractResult func(T) R,
) *EventStream[T, R] {
	return &EventStream[T, R]{
		ch:            make(chan T, 256),
		resultCh:      make(chan R, 1),
		isComplete:    isComplete,
		extractResult: extractResult,
		closed:        make(chan struct{}),
	}
}

func (s *EventStream[T, R]) Push(event T) {
	select {
	case <-s.closed:
		return
	default:
	}

	select {
	case s.ch <- event:
	default:
	}

	if s.isComplete(event) {
		r := s.extractResult(event)
		s.finish(&r)
	}
}

func (s *EventStream[T, R]) End(result *R) {
	s.finish(result)
}

func (s *EventStream[T, R]) Events() <-chan T {
	return s.ch
}

func (s *EventStream[T, R]) Result() <-chan R {
	return s.resultCh
}

func (s *EventStream[T, R]) finish(result *R) {
	s.closeOnce.Do(func() {
		if result != nil {
			s.resultCh <- *result
		}
		close(s.closed)
		close(s.ch)
		close(s.resultCh)
	})
}

type AssistantMessageEventStream = EventStream[AssistantMessageEvent, message.AssistantMessage]
