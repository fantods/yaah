package provider

import (
	"encoding/json"

	"github.com/fantods/yaah/internal/message"
)

type Context struct {
	SystemPrompt string            `json:"systemPrompt,omitempty"`
	Messages     []message.Message `json:"messages"`
	Tools        []Tool            `json:"tools"`
}

type messageRoleProxy struct {
	Role string `json:"role"`
}

func unmarshalMessages(data []byte) ([]message.Message, error) {
	if string(data) == "null" {
		return nil, nil
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	messages := make([]message.Message, 0, len(raw))
	for _, item := range raw {
		var proxy messageRoleProxy
		if err := json.Unmarshal(item, &proxy); err != nil {
			return nil, err
		}

		var msg message.Message
		switch proxy.Role {
		case "user":
			var m message.UserMessage
			if err := json.Unmarshal(item, &m); err != nil {
				return nil, err
			}
			msg = m
		case "assistant":
			var m message.AssistantMessage
			if err := json.Unmarshal(item, &m); err != nil {
				return nil, err
			}
			msg = m
		case "toolResult":
			var m message.ToolResultMessage
			if err := json.Unmarshal(item, &m); err != nil {
				return nil, err
			}
			msg = m
		default:
			messages = append(messages, nil)
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (c *Context) UnmarshalJSON(data []byte) error {
	var raw struct {
		SystemPrompt string          `json:"systemPrompt,omitempty"`
		Messages     json.RawMessage `json:"messages"`
		Tools        json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.SystemPrompt = raw.SystemPrompt

	msgs, err := unmarshalMessages(raw.Messages)
	if err != nil {
		return err
	}
	c.Messages = msgs

	if len(raw.Tools) > 0 && string(raw.Tools) != "null" {
		if err := json.Unmarshal(raw.Tools, &c.Tools); err != nil {
			return err
		}
	}

	return nil
}
