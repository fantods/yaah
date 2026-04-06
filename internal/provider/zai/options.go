package zai

import (
	"os"

	"github.com/fantods/yaah/internal/provider"
)

const DefaultBaseURL = "https://api.z.ai/api/coding/paas/v4"

const EnvAPIKey = "ZAI_API_KEY"

type ZaiOptions struct {
	EnableThinking bool
	ToolStream     bool
}

func ResolveAPIKey(opts *provider.StreamOptions) *string {
	if opts != nil && opts.APIKey != nil {
		return opts.APIKey
	}
	if key := os.Getenv(EnvAPIKey); key != "" {
		return &key
	}
	return nil
}
