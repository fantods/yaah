package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolConstruction(t *testing.T) {
	tool := Tool{
		Name:        "bash",
		Description: "Run a bash command",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"cmd":{"type":"string"}}}`),
	}

	assert.Equal(t, "bash", tool.Name)
	assert.Equal(t, "Run a bash command", tool.Description)
	assert.True(t, json.Valid(tool.Parameters))
}

func TestToolJSONRoundTrip(t *testing.T) {
	schema := `{"type":"object","properties":{"path":{"type":"string","description":"file path"}},"required":["path"]}`

	original := Tool{
		Name:        "read_file",
		Description: "Read a file from disk",
		Parameters:  json.RawMessage(schema),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Tool
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "read_file", decoded.Name)
	assert.Equal(t, "Read a file from disk", decoded.Description)
	assert.True(t, json.Valid(decoded.Parameters))

	var params map[string]any
	require.NoError(t, json.Unmarshal(decoded.Parameters, &params))
	assert.Equal(t, "object", params["type"])
}

func TestToolEmptyParameters(t *testing.T) {
	tool := Tool{
		Name:        "simple",
		Description: "A simple tool",
		Parameters:  json.RawMessage(`{}`),
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"parameters":{}`)
}
