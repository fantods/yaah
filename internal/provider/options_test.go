package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThinkingLevelConstants(t *testing.T) {
	assert.Equal(t, ThinkingLevel("minimal"), ThinkingLevelMinimal)
	assert.Equal(t, ThinkingLevel("low"), ThinkingLevelLow)
	assert.Equal(t, ThinkingLevel("medium"), ThinkingLevelMedium)
	assert.Equal(t, ThinkingLevel("high"), ThinkingLevelHigh)
	assert.Equal(t, ThinkingLevel("xhigh"), ThinkingLevelXHigh)
}

func TestThinkingLevelJSONRoundTrip(t *testing.T) {
	levels := []ThinkingLevel{
		ThinkingLevelMinimal,
		ThinkingLevelLow,
		ThinkingLevelMedium,
		ThinkingLevelHigh,
		ThinkingLevelXHigh,
	}

	for _, level := range levels {
		data, err := json.Marshal(level)
		require.NoError(t, err)

		var decoded ThinkingLevel
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, level, decoded)
	}
}

func TestThinkingBudgetsJSONRoundTrip(t *testing.T) {
	minimal := 1024
	low := 4096
	medium := 16384
	high := 65536

	original := ThinkingBudgets{
		Minimal: &minimal,
		Low:     &low,
		Medium:  &medium,
		High:    &high,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ThinkingBudgets
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.NotNil(t, decoded.Minimal)
	assert.Equal(t, 1024, *decoded.Minimal)
	require.NotNil(t, decoded.Low)
	assert.Equal(t, 4096, *decoded.Low)
	require.NotNil(t, decoded.Medium)
	assert.Equal(t, 16384, *decoded.Medium)
	require.NotNil(t, decoded.High)
	assert.Equal(t, 65536, *decoded.High)
}

func TestThinkingBudgetsOmitEmpty(t *testing.T) {
	original := ThinkingBudgets{}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	assert.Equal(t, "{}", string(data))
}

func TestThinkingBudgetsPartial(t *testing.T) {
	medium := 16384
	original := ThinkingBudgets{Medium: &medium}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ThinkingBudgets
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Nil(t, decoded.Minimal)
	assert.Nil(t, decoded.Low)
	require.NotNil(t, decoded.Medium)
	assert.Equal(t, 16384, *decoded.Medium)
	assert.Nil(t, decoded.High)
}

func TestCacheRetentionConstants(t *testing.T) {
	assert.Equal(t, CacheRetention("none"), CacheRetentionNone)
	assert.Equal(t, CacheRetention("short"), CacheRetentionShort)
	assert.Equal(t, CacheRetention("long"), CacheRetentionLong)
}

func TestTransportConstants(t *testing.T) {
	assert.Equal(t, Transport("sse"), TransportSSE)
	assert.Equal(t, Transport("websocket"), TransportWebsocket)
	assert.Equal(t, Transport("auto"), TransportAuto)
}

func TestStreamOptionsConstruction(t *testing.T) {
	opts := StreamOptions{
		Temperature:     floatPtr(0.7),
		MaxTokens:       intPtr(4096),
		APIKey:          stringPtr("sk-test"),
		Transport:       TransportAuto,
		CacheRetention:  CacheRetentionShort,
		SessionID:       "sess-123",
		Headers:         map[string]string{"X-Custom": "value"},
		MaxRetryDelayMs: intPtr(30000),
		Metadata:        map[string]any{"user_id": "u1"},
	}

	require.NotNil(t, opts.Temperature)
	assert.Equal(t, 0.7, *opts.Temperature)
	require.NotNil(t, opts.MaxTokens)
	assert.Equal(t, 4096, *opts.MaxTokens)
	require.NotNil(t, opts.APIKey)
	assert.Equal(t, "sk-test", *opts.APIKey)
	assert.Equal(t, TransportAuto, opts.Transport)
	assert.Equal(t, CacheRetentionShort, opts.CacheRetention)
	assert.Equal(t, "sess-123", opts.SessionID)
	assert.Equal(t, "value", opts.Headers["X-Custom"])
	require.NotNil(t, opts.MaxRetryDelayMs)
	assert.Equal(t, 30000, *opts.MaxRetryDelayMs)
	assert.Equal(t, "u1", opts.Metadata["user_id"])
}

func TestStreamOptionsJSONRoundTrip(t *testing.T) {
	original := StreamOptions{
		Temperature:     floatPtr(0.5),
		MaxTokens:       intPtr(2048),
		APIKey:          stringPtr("key"),
		Transport:       TransportSSE,
		CacheRetention:  CacheRetentionLong,
		SessionID:       "s1",
		Headers:         map[string]string{"Authorization": "Bearer x"},
		MaxRetryDelayMs: intPtr(60000),
		Metadata:        map[string]any{"k": "v"},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded StreamOptions
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.NotNil(t, decoded.Temperature)
	assert.Equal(t, 0.5, *decoded.Temperature)
	require.NotNil(t, decoded.MaxTokens)
	assert.Equal(t, 2048, *decoded.MaxTokens)
	assert.Equal(t, TransportSSE, decoded.Transport)
	assert.Equal(t, CacheRetentionLong, decoded.CacheRetention)
	assert.Equal(t, "s1", decoded.SessionID)
	assert.Equal(t, "Bearer x", decoded.Headers["Authorization"])
}

func TestStreamOptionsOmitEmpty(t *testing.T) {
	opts := StreamOptions{}

	data, err := json.Marshal(opts)
	require.NoError(t, err)

	assert.Equal(t, "{}", string(data))
}

func TestSimpleStreamOptionsConstruction(t *testing.T) {
	high := 65536
	opts := SimpleStreamOptions{
		StreamOptions: StreamOptions{
			Temperature: floatPtr(0.7),
			MaxTokens:   intPtr(8192),
		},
		Reasoning:       ThinkingLevelHigh,
		ThinkingBudgets: &ThinkingBudgets{High: &high},
	}

	assert.Equal(t, ThinkingLevelHigh, opts.Reasoning)
	require.NotNil(t, opts.ThinkingBudgets)
	require.NotNil(t, opts.ThinkingBudgets.High)
	assert.Equal(t, 65536, *opts.ThinkingBudgets.High)
	require.NotNil(t, opts.Temperature)
	assert.Equal(t, 0.7, *opts.Temperature)
}

func TestSimpleStreamOptionsJSONRoundTrip(t *testing.T) {
	medium := 16384
	original := SimpleStreamOptions{
		StreamOptions: StreamOptions{
			MaxTokens: intPtr(4096),
		},
		Reasoning:       ThinkingLevelMedium,
		ThinkingBudgets: &ThinkingBudgets{Medium: &medium},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded SimpleStreamOptions
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, ThinkingLevelMedium, decoded.Reasoning)
	require.NotNil(t, decoded.ThinkingBudgets)
	require.NotNil(t, decoded.ThinkingBudgets.Medium)
	assert.Equal(t, 16384, *decoded.ThinkingBudgets.Medium)
	require.NotNil(t, decoded.MaxTokens)
	assert.Equal(t, 4096, *decoded.MaxTokens)
}

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }
func stringPtr(v string) *string  { return &v }
