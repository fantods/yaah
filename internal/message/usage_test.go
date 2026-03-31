package message

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopReasonConstants(t *testing.T) {
	assert.Equal(t, StopReason("stop"), StopReasonStop)
	assert.Equal(t, StopReason("length"), StopReasonLength)
	assert.Equal(t, StopReason("toolUse"), StopReasonToolUse)
	assert.Equal(t, StopReason("aborted"), StopReasonAborted)
	assert.Equal(t, StopReason("error"), StopReasonError)
}

func TestStopReasonJSONRoundTrip(t *testing.T) {
	reasons := []StopReason{
		StopReasonStop,
		StopReasonLength,
		StopReasonToolUse,
		StopReasonAborted,
		StopReasonError,
	}

	for _, reason := range reasons {
		data, err := json.Marshal(reason)
		require.NoError(t, err)

		var decoded StopReason
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, reason, decoded)
	}
}

func TestCostJSONRoundTrip(t *testing.T) {
	original := Cost{
		Input:      3.0,
		Output:     15.0,
		CacheRead:  0.3,
		CacheWrite: 3.75,
		Total:      22.05,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Cost
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, 3.0, decoded.Input)
	assert.Equal(t, 15.0, decoded.Output)
	assert.Equal(t, 0.3, decoded.CacheRead)
	assert.Equal(t, 3.75, decoded.CacheWrite)
	assert.Equal(t, 22.05, decoded.Total)
}

func TestUsageJSONRoundTrip(t *testing.T) {
	original := Usage{
		Input:       1000,
		Output:      500,
		CacheRead:   200,
		CacheWrite:  300,
		TotalTokens: 2000,
		Cost: Cost{
			Input:      0.003,
			Output:     0.0075,
			CacheRead:  0.00006,
			CacheWrite: 0.001125,
			Total:      0.011685,
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Usage
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, int64(1000), decoded.Input)
	assert.Equal(t, int64(500), decoded.Output)
	assert.Equal(t, int64(200), decoded.CacheRead)
	assert.Equal(t, int64(300), decoded.CacheWrite)
	assert.Equal(t, int64(2000), decoded.TotalTokens)
	assert.Equal(t, 0.003, decoded.Cost.Input)
	assert.Equal(t, 0.0075, decoded.Cost.Output)
	assert.Equal(t, 0.00006, decoded.Cost.CacheRead)
	assert.Equal(t, 0.001125, decoded.Cost.CacheWrite)
	assert.Equal(t, 0.011685, decoded.Cost.Total)
}

func TestUsageZeroValues(t *testing.T) {
	var u Usage
	assert.Equal(t, int64(0), u.Input)
	assert.Equal(t, int64(0), u.Output)
	assert.Equal(t, int64(0), u.CacheRead)
	assert.Equal(t, int64(0), u.CacheWrite)
	assert.Equal(t, int64(0), u.TotalTokens)
	assert.Equal(t, 0.0, u.Cost.Input)
	assert.Equal(t, 0.0, u.Cost.Output)
	assert.Equal(t, 0.0, u.Cost.CacheRead)
	assert.Equal(t, 0.0, u.Cost.CacheWrite)
	assert.Equal(t, 0.0, u.Cost.Total)
}

func TestUsageAdd(t *testing.T) {
	a := Usage{
		Input:       100,
		Output:      50,
		CacheRead:   20,
		CacheWrite:  30,
		TotalTokens: 200,
		Cost:        Cost{Input: 1.0, Output: 2.0, CacheRead: 0.5, CacheWrite: 0.75, Total: 4.25},
	}

	b := Usage{
		Input:       200,
		Output:      75,
		CacheRead:   10,
		CacheWrite:  15,
		TotalTokens: 300,
		Cost:        Cost{Input: 2.0, Output: 3.0, CacheRead: 0.25, CacheWrite: 0.5, Total: 5.75},
	}

	combined := a.Add(b)

	assert.Equal(t, int64(300), combined.Input)
	assert.Equal(t, int64(125), combined.Output)
	assert.Equal(t, int64(30), combined.CacheRead)
	assert.Equal(t, int64(45), combined.CacheWrite)
	assert.Equal(t, int64(500), combined.TotalTokens)
	assert.Equal(t, 3.0, combined.Cost.Input)
	assert.Equal(t, 5.0, combined.Cost.Output)
	assert.Equal(t, 0.75, combined.Cost.CacheRead)
	assert.Equal(t, 1.25, combined.Cost.CacheWrite)
	assert.Equal(t, 10.0, combined.Cost.Total)
}

func TestUsageAddDoesNotMutateReceiver(t *testing.T) {
	a := Usage{Input: 100, Output: 50, TotalTokens: 150}
	b := Usage{Input: 200, Output: 75, TotalTokens: 275}

	_ = a.Add(b)

	assert.Equal(t, int64(100), a.Input)
	assert.Equal(t, int64(50), a.Output)
	assert.Equal(t, int64(150), a.TotalTokens)
}
