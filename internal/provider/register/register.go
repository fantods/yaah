package register

import (
	"github.com/fantods/yaah/internal/provider"
	"github.com/fantods/yaah/internal/provider/anthropic"
	"github.com/fantods/yaah/internal/provider/openai"
	"github.com/fantods/yaah/internal/provider/zai"
)

func Builtins() {
	provider.Register(provider.Provider{
		API:          "anthropic-messages",
		Stream:       anthropic.Stream,
		StreamSimple: anthropic.StreamSimple,
	})
	provider.Register(provider.Provider{
		API:          "openai-completions",
		Stream:       openai.Stream,
		StreamSimple: openai.StreamSimple,
	})
	provider.Register(provider.Provider{
		API:          "zai",
		Stream:       zai.Stream,
		StreamSimple: zai.StreamSimple,
	})
}
