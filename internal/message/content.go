package message

type ContentBlock interface {
	contentBlock()
}

type TextContent struct {
	Type          string `json:"type"`
	Text          string `json:"text"`
	TextSignature string `json:"textSignature,omitempty"`
}

func (TextContent) contentBlock() {}

type ThinkingContent struct {
	Type              string `json:"type"`
	Thinking          string `json:"thinking"`
	ThinkingSignature string `json:"thinkingSignature,omitempty"`
	Redacted          bool   `json:"redacted,omitempty"`
}

func (ThinkingContent) contentBlock() {}

type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MIMEType string `json:"mimeType"`
}

func (ImageContent) contentBlock() {}

type ToolCall struct {
	Type             string         `json:"type"`
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Arguments        map[string]any `json:"arguments"`
	ThoughtSignature string         `json:"thoughtSignature,omitempty"`
}

func (ToolCall) contentBlock() {}
