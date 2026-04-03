package openai

import (
	"strings"

	"github.com/fantods/yaah/internal/provider"
)

type ThinkingFormat string

const (
	ThinkingFormatOpenAI           ThinkingFormat = "openai"
	ThinkingFormatOpenRouter       ThinkingFormat = "openrouter"
	ThinkingFormatZAI              ThinkingFormat = "zai"
	ThinkingFormatQwen             ThinkingFormat = "qwen"
	ThinkingFormatQwenChatTemplate ThinkingFormat = "qwen-chat-template"
)

type MaxTokensField string

const (
	MaxTokensFieldCompletion MaxTokensField = "max_completion_tokens"
	MaxTokensFieldLegacy     MaxTokensField = "max_tokens"
)

type Compat struct {
	SupportsStore            bool
	SupportsDeveloperRole    bool
	SupportsReasoningEffort  bool
	SupportsUsageInStreaming bool
	SupportsStrictMode       bool
	MaxTokensField           MaxTokensField
	RequiresToolResultName   bool
	RequiresThinkingAsText   bool
	ThinkingFormat           ThinkingFormat
	ReasoningEffortMap       map[provider.ThinkingLevel]string
}

func DefaultCompat() Compat {
	return Compat{
		SupportsStore:            true,
		SupportsDeveloperRole:    true,
		SupportsReasoningEffort:  true,
		SupportsUsageInStreaming: true,
		SupportsStrictMode:       true,
		MaxTokensField:           MaxTokensFieldCompletion,
		ThinkingFormat:           ThinkingFormatOpenAI,
	}
}

func DetectCompat(model provider.Model) Compat {
	compat := DefaultCompat()

	baseURL := strings.ToLower(model.BaseURL)
	providerName := strings.ToLower(model.Provider)

	if providerName == "zai" || strings.Contains(baseURL, "api.z.ai") {
		compat.SupportsDeveloperRole = false
		compat.SupportsStore = false
		compat.SupportsReasoningEffort = false
		compat.ThinkingFormat = ThinkingFormatZAI
	}

	if strings.Contains(baseURL, "cerebras.ai") {
		compat.SupportsStore = false
		compat.SupportsDeveloperRole = false
	}

	if strings.Contains(baseURL, "api.x.ai") {
		compat.SupportsReasoningEffort = false
	}

	if strings.Contains(baseURL, "chutes.ai") {
		compat.MaxTokensField = MaxTokensFieldLegacy
	}

	if strings.Contains(baseURL, "groq.com") {
		compat.SupportsStore = false
		compat.SupportsDeveloperRole = false
		compat.ReasoningEffortMap = map[provider.ThinkingLevel]string{
			provider.ThinkingLevelMinimal: "default",
			provider.ThinkingLevelLow:     "default",
			provider.ThinkingLevelMedium:  "default",
			provider.ThinkingLevelHigh:    "default",
			provider.ThinkingLevelXHigh:   "default",
		}
	}

	if providerName == "openrouter" || strings.Contains(baseURL, "openrouter.ai") {
		compat.ThinkingFormat = ThinkingFormatOpenRouter
	}

	if model.Compat != nil {
		if c, ok := model.Compat.(Compat); ok {
			return c
		}
	}

	return compat
}

type OpenAIOptions struct {
	provider.StreamOptions
	ToolChoice      any
	ReasoningEffort provider.ThinkingLevel
	Compat          Compat
}
