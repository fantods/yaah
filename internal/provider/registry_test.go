package provider

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryRegisterAndLookup(t *testing.T) {
	ResetRegistry()

	p := Provider{API: "test-api"}
	Register(p)

	got, ok := Lookup("test-api")
	require.True(t, ok)
	assert.Equal(t, "test-api", got.API)
}

func TestRegistryLookupNotFound(t *testing.T) {
	ResetRegistry()

	_, ok := Lookup("nonexistent")
	assert.False(t, ok)
}

func TestRegistryOverwrite(t *testing.T) {
	ResetRegistry()

	Register(Provider{API: "x", Stream: func(Model, Context, *StreamOptions) *AssistantMessageEventStream { return nil }})
	Register(Provider{API: "x", Stream: func(Model, Context, *StreamOptions) *AssistantMessageEventStream { return nil }})

	got, ok := Lookup("x")
	require.True(t, ok)
	assert.NotNil(t, got.Stream)
}

func TestRegistryConcurrentAccess(t *testing.T) {
	ResetRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			Register(Provider{API: "concurrent"})
		}(i)
		go func(n int) {
			defer wg.Done()
			Lookup("concurrent")
		}(i)
	}
	wg.Wait()
}

func TestRegistryMultipleProviders(t *testing.T) {
	ResetRegistry()

	Register(Provider{API: "anthropic-messages"})
	Register(Provider{API: "openai-completions"})
	Register(Provider{API: "zai"})

	for _, api := range []string{"anthropic-messages", "openai-completions", "zai"} {
		got, ok := Lookup(api)
		require.True(t, ok, "expected to find %q", api)
		assert.Equal(t, api, got.API)
	}
}
