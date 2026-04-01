package provider

import "github.com/fantods/yaah/internal/message"

type AssistantMessageEvent interface{ assistantEvent() }

type EventStart struct {
	Partial *message.AssistantMessage
}

func (EventStart) assistantEvent() {}

type EventTextStart struct {
	Partial      *message.AssistantMessage
	ContentIndex int
}

func (EventTextStart) assistantEvent() {}

type EventTextDelta struct {
	Partial      *message.AssistantMessage
	ContentIndex int
	Delta        string
}

func (EventTextDelta) assistantEvent() {}

type EventTextEnd struct {
	Partial      *message.AssistantMessage
	ContentIndex int
	Content      string
}

func (EventTextEnd) assistantEvent() {}

type EventThinkingStart struct {
	Partial      *message.AssistantMessage
	ContentIndex int
}

func (EventThinkingStart) assistantEvent() {}

type EventThinkingDelta struct {
	Partial      *message.AssistantMessage
	ContentIndex int
	Delta        string
}

func (EventThinkingDelta) assistantEvent() {}

type EventThinkingEnd struct {
	Partial      *message.AssistantMessage
	ContentIndex int
	Content      string
}

func (EventThinkingEnd) assistantEvent() {}

type EventToolCallStart struct {
	Partial      *message.AssistantMessage
	ContentIndex int
}

func (EventToolCallStart) assistantEvent() {}

type EventToolCallDelta struct {
	Partial      *message.AssistantMessage
	ContentIndex int
	Delta        string
}

func (EventToolCallDelta) assistantEvent() {}

type EventToolCallEnd struct {
	Partial      *message.AssistantMessage
	ContentIndex int
	ToolCall     message.ToolCall
}

func (EventToolCallEnd) assistantEvent() {}

type EventDone struct {
	Reason  message.StopReason
	Message message.AssistantMessage
}

func (EventDone) assistantEvent() {}

type EventError struct {
	Reason  message.StopReason
	Message message.AssistantMessage
}

func (EventError) assistantEvent() {}
