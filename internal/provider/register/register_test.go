package register_test

import (
	"testing"

	"github.com/fantods/yaah/internal/provider"
	_ "github.com/fantods/yaah/internal/provider/anthropic"
	_ "github.com/fantods/yaah/internal/provider/openai"
	_ "github.com/fantods/yaah/internal/provider/zai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvidersAutoRegistered(t *testing.T) {
	for _, api := range []string{"anthropic-messages", "openai-completions", "zai"} {
		got, ok := provider.Lookup(api)
		require.True(t, ok, "expected to find %q", api)
		assert.Equal(t, api, got.API)
		assert.NotNil(t, got.Stream, "%q Stream should not be nil", api)
		assert.NotNil(t, got.StreamSimple, "%q StreamSimple should not be nil", api)
	}
}
