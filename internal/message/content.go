package message

import "encoding/json"

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

func ExtractText(blocks []ContentBlock) string {
	var text string
	for _, block := range blocks {
		if tc, ok := block.(TextContent); ok {
			text += tc.Text
		}
	}
	return text
}

type contentBlockProxy struct {
	Type string `json:"type"`
}

func unmarshalContentBlocks(data []byte) ([]ContentBlock, error) {
	if string(data) == "null" {
		return nil, nil
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	blocks := make([]ContentBlock, 0, len(raw))
	for _, item := range raw {
		var proxy contentBlockProxy
		if err := json.Unmarshal(item, &proxy); err != nil {
			return nil, err
		}

		var block ContentBlock
		switch proxy.Type {
		case "text":
			var tc TextContent
			if err := json.Unmarshal(item, &tc); err != nil {
				return nil, err
			}
			block = tc
		case "thinking":
			var tc ThinkingContent
			if err := json.Unmarshal(item, &tc); err != nil {
				return nil, err
			}
			block = tc
		case "image":
			var ic ImageContent
			if err := json.Unmarshal(item, &ic); err != nil {
				return nil, err
			}
			block = ic
		case "toolCall":
			var tc ToolCall
			if err := json.Unmarshal(item, &tc); err != nil {
				return nil, err
			}
			block = tc
		default:
			blocks = append(blocks, nil)
			continue
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}
