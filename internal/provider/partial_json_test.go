package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartialJSONEmpty(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse("")
	assert.True(t, ok)
	assert.Equal(t, map[string]any{}, result)
}

func TestPartialJSONCompleteObject(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"key":"value"}`)
	assert.True(t, ok)
	assert.Equal(t, "value", result["key"])
}

func TestPartialJSONIncremental(t *testing.T) {
	p := NewPartialJSONParser()

	_, ok := p.Parse(`{"pa`)
	assert.False(t, ok)

	_, ok = p.Parse(`th":`)
	assert.False(t, ok)

	result, ok := p.Parse(`"/tmp/test.txt"}`)
	assert.True(t, ok)
	assert.Equal(t, "/tmp/test.txt", result["path"])
}

func TestPartialJSONMultipleKeys(t *testing.T) {
	p := NewPartialJSONParser()

	p.Parse(`{"na`)
	p.Parse(`me":`)
	p.Parse(`"bash",`)
	p.Parse(`"arg`)
	p.Parse(`s":`)
	result, ok := p.Parse(`{"x":1}}`)

	assert.True(t, ok)
	assert.Equal(t, "bash", result["name"])
	args, ok := result["args"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), args["x"])
}

func TestPartialJSONArrayValue(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"files":["a.txt","b.txt"]}`)

	assert.True(t, ok)
	files, ok := result["files"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"a.txt", "b.txt"}, files)
}

func TestPartialJSONNumericValue(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"count":42,"ratio":3.14}`)

	assert.True(t, ok)
	assert.Equal(t, float64(42), result["count"])
	assert.Equal(t, 3.14, result["ratio"])
}

func TestPartialJSONBooleanValue(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"verbose":true,"dry":false}`)

	assert.True(t, ok)
	assert.Equal(t, true, result["verbose"])
	assert.Equal(t, false, result["dry"])
}

func TestPartialJSONNullValue(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"val":null}`)

	assert.True(t, ok)
	assert.Nil(t, result["val"])
}

func TestPartialJSONReset(t *testing.T) {
	p := NewPartialJSONParser()
	p.Parse(`{"a":1}`)

	p.Reset()
	_, ok := p.Parse(`{"b":2}`)
	assert.True(t, ok)
}

func TestPartialJSONWhitespaceHandling(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{ "key" : "value" }`)

	assert.True(t, ok)
	assert.Equal(t, "value", result["key"])
}

func TestPartialJSONEscapedString(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"path":"/tmp/hello world.txt","msg":"line1\nline2"}`)

	assert.True(t, ok)
	assert.Equal(t, "/tmp/hello world.txt", result["path"])
	assert.Equal(t, "line1\nline2", result["msg"])
}

func TestPartialJSONTrailingComma(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"a":1,}`)

	assert.True(t, ok)
	assert.Equal(t, float64(1), result["a"])
}

func TestPartialJSONNestedObject(t *testing.T) {
	p := NewPartialJSONParser()
	result, ok := p.Parse(`{"outer":{"inner":"deep"}}`)

	assert.True(t, ok)
	outer, ok := result["outer"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "deep", outer["inner"])
}

func TestPartialJSONGetBuffer(t *testing.T) {
	p := NewPartialJSONParser()
	p.Parse(`{"key`)
	assert.Equal(t, `{"key`, p.Buffer())
}
