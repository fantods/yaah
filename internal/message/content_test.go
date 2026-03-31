package message

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextContentImplementsContentBlock(t *testing.T) {
	var _ ContentBlock = TextContent{}
}

func TestThinkingContentImplementsContentBlock(t *testing.T) {
	var _ ContentBlock = ThinkingContent{}
}

func TestImageContentImplementsContentBlock(t *testing.T) {
	var _ ContentBlock = ImageContent{}
}

func TestToolCallImplementsContentBlock(t *testing.T) {
	var _ ContentBlock = ToolCall{}
}

func TestTextContentJSONRoundTrip(t *testing.T) {
	original := TextContent{
		Type:          "text",
		Text:          "hello world",
		TextSignature: "sig-123",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded TextContent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "text", decoded.Type)
	assert.Equal(t, "hello world", decoded.Text)
	assert.Equal(t, "sig-123", decoded.TextSignature)
}

func TestTextContentSignatureOmitted(t *testing.T) {
	original := TextContent{
		Type: "text",
		Text: "hello",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "textSignature")

	var decoded TextContent
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "", decoded.TextSignature)
}

func TestThinkingContentJSONRoundTrip(t *testing.T) {
	original := ThinkingContent{
		Type:              "thinking",
		Thinking:          "let me think...",
		ThinkingSignature: "tsig-456",
		Redacted:          true,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ThinkingContent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "thinking", decoded.Type)
	assert.Equal(t, "let me think...", decoded.Thinking)
	assert.Equal(t, "tsig-456", decoded.ThinkingSignature)
	assert.True(t, decoded.Redacted)
}

func TestThinkingContentRedactedOmitted(t *testing.T) {
	original := ThinkingContent{
		Type:     "thinking",
		Thinking: "hmm",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "redacted")
}

func TestImageContentJSONRoundTrip(t *testing.T) {
	original := ImageContent{
		Type:     "image",
		Data:     "aGVsbG8=",
		MIMEType: "image/png",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ImageContent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "image", decoded.Type)
	assert.Equal(t, "aGVsbG8=", decoded.Data)
	assert.Equal(t, "image/png", decoded.MIMEType)
}

func TestToolCallJSONRoundTrip(t *testing.T) {
	original := ToolCall{
		Type:             "toolCall",
		ID:               "call_abc123",
		Name:             "read_file",
		Arguments:        map[string]any{"path": "/tmp/test.txt"},
		ThoughtSignature: "thsig-789",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ToolCall
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "toolCall", decoded.Type)
	assert.Equal(t, "call_abc123", decoded.ID)
	assert.Equal(t, "read_file", decoded.Name)
	assert.Equal(t, "/tmp/test.txt", decoded.Arguments["path"])
	assert.Equal(t, "thsig-789", decoded.ThoughtSignature)
}

func TestToolCallThoughtSignatureOmitted(t *testing.T) {
	original := ToolCall{
		Type:      "toolCall",
		ID:        "call_1",
		Name:      "bash",
		Arguments: map[string]any{},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "thoughtSignature")
}

func TestContentTypeSwitch(t *testing.T) {
	blocks := []ContentBlock{
		TextContent{Type: "text", Text: "hi"},
		ThinkingContent{Type: "thinking", Thinking: "hmm"},
		ImageContent{Type: "image", Data: "ZGF0YQ==", MIMEType: "image/jpeg"},
		ToolCall{Type: "toolCall", ID: "c1", Name: "run", Arguments: map[string]any{}},
	}

	assert.Equal(t, "text", blocks[0].(TextContent).Type)
	assert.Equal(t, "thinking", blocks[1].(ThinkingContent).Type)
	assert.Equal(t, "image", blocks[2].(ImageContent).Type)
	assert.Equal(t, "toolCall", blocks[3].(ToolCall).Type)

	for _, block := range blocks {
		switch b := block.(type) {
		case TextContent:
			assert.Equal(t, "text", b.Type)
		case ThinkingContent:
			assert.Equal(t, "thinking", b.Type)
		case ImageContent:
			assert.Equal(t, "image", b.Type)
		case ToolCall:
			assert.Equal(t, "toolCall", b.Type)
		default:
			t.Fatalf("unexpected type: %T", b)
		}
	}
}

func TestToolCallArgumentsMap(t *testing.T) {
	tc := ToolCall{
		Type:      "toolCall",
		ID:        "call_1",
		Name:      "bash",
		Arguments: map[string]any{},
	}
	assert.NotNil(t, tc.Arguments)

	tc.Arguments["key"] = "value"
	assert.Equal(t, "value", tc.Arguments["key"])
}
