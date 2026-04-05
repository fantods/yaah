package agent

import (
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

type AgentEvent interface{ agentEvent() }

type AgentStartEvent struct{}

func (AgentStartEvent) agentEvent() {}

type AgentEndEvent struct {
	Messages []message.Message
}

func (AgentEndEvent) agentEvent() {}

type TurnStartEvent struct{}

func (TurnStartEvent) agentEvent() {}

type TurnEndEvent struct {
	Message     message.Message
	ToolResults []message.ToolResultMessage
}

func (TurnEndEvent) agentEvent() {}

type MessageStartEvent struct {
	Message message.Message
}

func (MessageStartEvent) agentEvent() {}

type MessageUpdateEvent struct {
	Message               message.Message
	AssistantMessageEvent provider.AssistantMessageEvent
}

func (MessageUpdateEvent) agentEvent() {}

type MessageEndEvent struct {
	Message message.Message
}

func (MessageEndEvent) agentEvent() {}

type ToolExecStartEvent struct {
	ToolCallID string
	ToolName   string
	Args       map[string]any
}

func (ToolExecStartEvent) agentEvent() {}

type ToolExecUpdateEvent struct {
	ToolCallID    string
	ToolName      string
	Args          map[string]any
	PartialResult any
}

func (ToolExecUpdateEvent) agentEvent() {}

type ToolExecEndEvent struct {
	ToolCallID string
	ToolName   string
	Result     any
	IsError    bool
}

func (ToolExecEndEvent) agentEvent() {}
