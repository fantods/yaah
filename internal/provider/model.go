package provider

import "fmt"

type KnownProvider string

const (
	KnownProviderAnthropic KnownProvider = "anthropic"
	KnownProviderOpenAI    KnownProvider = "openai"
	KnownProviderZAI       KnownProvider = "zai"
)

type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

type Model struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	API           string            `json:"api"`
	Provider      string            `json:"provider"`
	BaseURL       string            `json:"baseUrl"`
	Reasoning     bool              `json:"reasoning"`
	Input         []string          `json:"input"`
	Cost          ModelCost         `json:"cost"`
	ContextWindow int               `json:"contextWindow"`
	MaxTokens     int               `json:"maxTokens"`
	Headers       map[string]string `json:"headers,omitempty"`
	Compat        any               `json:"compat,omitempty"`
}

var catalog = []Model{
	{
		ID:            "claude-sonnet-4-20250514",
		Name:          "Claude Sonnet 4",
		API:           "anthropic-messages",
		Provider:      "anthropic",
		MaxTokens:     8192,
		ContextWindow: 200000,
		Input:         []string{"text", "image"},
	},
	{
		ID:            "claude-haiku-4-20250414",
		Name:          "Claude Haiku 4",
		API:           "anthropic-messages",
		Provider:      "anthropic",
		MaxTokens:     8192,
		ContextWindow: 200000,
		Input:         []string{"text", "image"},
	},
	{
		ID:            "gpt-4.1",
		Name:          "GPT-4.1",
		API:           "openai-completions",
		Provider:      "openai",
		MaxTokens:     16384,
		ContextWindow: 1047576,
		Input:         []string{"text", "image"},
	},
	{
		ID:            "gpt-4.1-mini",
		Name:          "GPT-4.1 Mini",
		API:           "openai-completions",
		Provider:      "openai",
		MaxTokens:     16384,
		ContextWindow: 1047576,
		Input:         []string{"text", "image"},
	},
	{
		ID:            "gpt-4.1-nano",
		Name:          "GPT-4.1 Nano",
		API:           "openai-completions",
		Provider:      "openai",
		MaxTokens:     16384,
		ContextWindow: 1047576,
		Input:         []string{"text", "image"},
	},
	{
		ID:            "o3",
		Name:          "o3",
		API:           "openai-completions",
		Provider:      "openai",
		Reasoning:     true,
		MaxTokens:     32768,
		ContextWindow: 200000,
		Input:         []string{"text", "image"},
	},
	{
		ID:            "o4-mini",
		Name:          "o4-mini",
		API:           "openai-completions",
		Provider:      "openai",
		Reasoning:     true,
		MaxTokens:     32768,
		ContextWindow: 200000,
		Input:         []string{"text", "image"},
	},
}

func Catalog() []Model {
	return catalog
}

func LookupModel(id string) (Model, error) {
	for _, m := range catalog {
		if m.ID == id {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("unknown model %q (use --list-models to see available models)", id)
}
