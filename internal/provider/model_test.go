package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelCostJSONRoundTrip(t *testing.T) {
	original := ModelCost{
		Input:      3.0,
		Output:     15.0,
		CacheRead:  0.3,
		CacheWrite: 3.75,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ModelCost
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, 3.0, decoded.Input)
	assert.Equal(t, 15.0, decoded.Output)
	assert.Equal(t, 0.3, decoded.CacheRead)
	assert.Equal(t, 3.75, decoded.CacheWrite)
}

func TestModelCostZeroValues(t *testing.T) {
	var mc ModelCost
	assert.Equal(t, 0.0, mc.Input)
	assert.Equal(t, 0.0, mc.Output)
	assert.Equal(t, 0.0, mc.CacheRead)
	assert.Equal(t, 0.0, mc.CacheWrite)
}

func TestModelConstruction(t *testing.T) {
	m := Model{
		ID:            "claude-sonnet-4-20250514",
		Name:          "Claude Sonnet 4",
		API:           "anthropic-messages",
		Provider:      "anthropic",
		BaseURL:       "https://api.anthropic.com",
		Reasoning:     true,
		Input:         []string{"text", "image"},
		Cost:          ModelCost{Input: 3.0, Output: 15.0, CacheRead: 0.3, CacheWrite: 3.75},
		ContextWindow: 200000,
		MaxTokens:     8192,
		Headers:       map[string]string{"anthropic-version": "2023-06-01"},
	}

	assert.Equal(t, "claude-sonnet-4-20250514", m.ID)
	assert.Equal(t, "Claude Sonnet 4", m.Name)
	assert.Equal(t, "anthropic-messages", m.API)
	assert.Equal(t, "anthropic", m.Provider)
	assert.True(t, m.Reasoning)
	assert.Equal(t, []string{"text", "image"}, m.Input)
	assert.Equal(t, 200000, m.ContextWindow)
	assert.Equal(t, 8192, m.MaxTokens)
	assert.Equal(t, 3.0, m.Cost.Input)
	assert.Equal(t, "2023-06-01", m.Headers["anthropic-version"])
}

func TestModelJSONRoundTrip(t *testing.T) {
	original := Model{
		ID:            "gpt-4o",
		Name:          "GPT-4o",
		API:           "openai-completions",
		Provider:      "openai",
		BaseURL:       "https://api.openai.com/v1",
		Reasoning:     false,
		Input:         []string{"text", "image"},
		Cost:          ModelCost{Input: 5.0, Output: 15.0},
		ContextWindow: 128000,
		MaxTokens:     16384,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Model
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.API, decoded.API)
	assert.Equal(t, original.Provider, decoded.Provider)
	assert.Equal(t, original.BaseURL, decoded.BaseURL)
	assert.Equal(t, original.Reasoning, decoded.Reasoning)
	assert.Equal(t, original.Input, decoded.Input)
	assert.Equal(t, 5.0, decoded.Cost.Input)
	assert.Equal(t, 15.0, decoded.Cost.Output)
	assert.Equal(t, original.ContextWindow, decoded.ContextWindow)
	assert.Equal(t, original.MaxTokens, decoded.MaxTokens)
}

func TestModelOmitEmptyFields(t *testing.T) {
	m := Model{
		ID:       "test-model",
		Name:     "Test",
		API:      "test",
		Provider: "test",
		BaseURL:  "https://example.com",
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "headers")
	assert.NotContains(t, string(data), "compat")
}

func TestModelWithCompat(t *testing.T) {
	m := Model{
		ID:       "test",
		Name:     "Test",
		API:      "openai-completions",
		Provider: "test",
		BaseURL:  "https://example.com",
		Compat: map[string]any{
			"supportsStore":          true,
			"supportsDeveloperRole":  false,
			"requiresThinkingAsText": true,
			"maxTokensField":         "max_completion_tokens",
		},
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var decoded Model
	require.NoError(t, json.Unmarshal(data, &decoded))

	compatMap, ok := decoded.Compat.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, compatMap["supportsStore"])
}
