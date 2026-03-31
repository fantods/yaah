package message

import "encoding/json"

type Message interface{ messageUnion() }

type UserMessage struct {
	Role      string         `json:"role"`
	Content   []ContentBlock `json:"content"`
	Timestamp int64          `json:"timestamp"`
}

func (UserMessage) messageUnion() {}

func (m *UserMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role      string          `json:"role"`
		Content   json.RawMessage `json:"content"`
		Timestamp int64           `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	blocks, err := unmarshalContentBlocks(raw.Content)
	if err != nil {
		return err
	}

	m.Role = raw.Role
	m.Content = blocks
	m.Timestamp = raw.Timestamp
	return nil
}

type AssistantMessage struct {
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	API          string         `json:"api"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model"`
	ResponseID   string         `json:"responseId,omitempty"`
	Usage        Usage          `json:"usage"`
	StopReason   StopReason     `json:"stopReason"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	Timestamp    int64          `json:"timestamp"`
}

func (AssistantMessage) messageUnion() {}

func (m *AssistantMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role         string          `json:"role"`
		Content      json.RawMessage `json:"content"`
		API          string          `json:"api"`
		Provider     string          `json:"provider"`
		Model        string          `json:"model"`
		ResponseID   string          `json:"responseId,omitempty"`
		Usage        Usage           `json:"usage"`
		StopReason   StopReason      `json:"stopReason"`
		ErrorMessage string          `json:"errorMessage,omitempty"`
		Timestamp    int64           `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	blocks, err := unmarshalContentBlocks(raw.Content)
	if err != nil {
		return err
	}

	m.Role = raw.Role
	m.Content = blocks
	m.API = raw.API
	m.Provider = raw.Provider
	m.Model = raw.Model
	m.ResponseID = raw.ResponseID
	m.Usage = raw.Usage
	m.StopReason = raw.StopReason
	m.ErrorMessage = raw.ErrorMessage
	m.Timestamp = raw.Timestamp
	return nil
}

type ToolResultMessage struct {
	Role       string         `json:"role"`
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Content    []ContentBlock `json:"content"`
	Details    any            `json:"details,omitempty"`
	IsError    bool           `json:"isError"`
	Timestamp  int64          `json:"timestamp"`
}

func (ToolResultMessage) messageUnion() {}

func (m *ToolResultMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role       string          `json:"role"`
		ToolCallID string          `json:"toolCallId"`
		ToolName   string          `json:"toolName"`
		Content    json.RawMessage `json:"content"`
		Details    any             `json:"details,omitempty"`
		IsError    bool            `json:"isError"`
		Timestamp  int64           `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	blocks, err := unmarshalContentBlocks(raw.Content)
	if err != nil {
		return err
	}

	m.Role = raw.Role
	m.ToolCallID = raw.ToolCallID
	m.ToolName = raw.ToolName
	m.Content = blocks
	m.Details = raw.Details
	m.IsError = raw.IsError
	m.Timestamp = raw.Timestamp
	return nil
}
