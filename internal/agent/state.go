package agent

import (
	"sync"

	"github.com/fantods/yaah/internal/message"
)

type AgentState struct {
	mu               sync.RWMutex
	Streaming        bool
	Messages         []message.Message
	PendingToolCalls []AgentToolCall
	Error            error
	Turn             int
}

func NewAgentState() *AgentState {
	return &AgentState{
		Messages:         []message.Message{},
		PendingToolCalls: []AgentToolCall{},
	}
}

func (s *AgentState) SetStreaming(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Streaming = v
}

func (s *AgentState) IsStreaming() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Streaming
}

func (s *AgentState) AddMessage(msg message.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, msg)
}

func (s *AgentState) GetMessages() []message.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]message.Message, len(s.Messages))
	copy(cp, s.Messages)
	return cp
}

func (s *AgentState) SetPendingToolCalls(calls []AgentToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PendingToolCalls = calls
}

func (s *AgentState) GetPendingToolCalls() []AgentToolCall {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]AgentToolCall, len(s.PendingToolCalls))
	copy(cp, s.PendingToolCalls)
	return cp
}

func (s *AgentState) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
}

func (s *AgentState) GetError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Error
}

func (s *AgentState) IncrementTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Turn++
}

func (s *AgentState) GetTurn() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Turn
}
