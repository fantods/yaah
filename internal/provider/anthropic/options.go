package anthropic

import (
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/fantods/yaah/internal/provider"
)

type AnthropicOptions struct {
	StealthMode    bool
	CacheRetention provider.CacheRetention
}

func (o AnthropicOptions) BuildCacheControl() *anthropic.CacheControlEphemeralParam {
	switch o.CacheRetention {
	case provider.CacheRetentionShort, provider.CacheRetentionLong:
		return &anthropic.CacheControlEphemeralParam{}
	default:
		return nil
	}
}

func (o AnthropicOptions) BuildSystemPrompt(prompt string) []anthropic.TextBlockParam {
	if prompt == "" {
		return nil
	}

	block := anthropic.TextBlockParam{
		Text: prompt,
	}

	if cc := o.BuildCacheControl(); cc != nil {
		block.CacheControl = *cc
	}

	return []anthropic.TextBlockParam{block}
}

func StealthToolRename(name string) string {
	const suffix = "_tool"
	if strings.HasSuffix(name, suffix) {
		return name
	}
	return name + suffix
}

func ApplyStealthMode(tools []provider.Tool) []provider.Tool {
	result := make([]provider.Tool, len(tools))
	for i, tool := range tools {
		result[i] = tool
		result[i].Name = StealthToolRename(tool.Name)
	}
	return result
}
