package anthropic

import (
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/fantods/yaah/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicOptionsDefaults(t *testing.T) {
	opts := AnthropicOptions{}
	assert.False(t, opts.StealthMode)
}

func TestAnthropicOptionsWithStealthMode(t *testing.T) {
	opts := AnthropicOptions{StealthMode: true}
	assert.True(t, opts.StealthMode)
}

func TestAnthropicOptionsWithCacheRetention(t *testing.T) {
	opts := AnthropicOptions{CacheRetention: provider.CacheRetentionShort}
	assert.Equal(t, provider.CacheRetentionShort, opts.CacheRetention)
}

func TestBuildCacheControlNone(t *testing.T) {
	opts := AnthropicOptions{CacheRetention: provider.CacheRetentionNone}
	cc := opts.BuildCacheControl()
	assert.Nil(t, cc)
}

func TestBuildCacheControlShort(t *testing.T) {
	opts := AnthropicOptions{CacheRetention: provider.CacheRetentionShort}
	cc := opts.BuildCacheControl()
	require.NotNil(t, cc)
}

func TestBuildCacheControlLong(t *testing.T) {
	opts := AnthropicOptions{CacheRetention: provider.CacheRetentionLong}
	cc := opts.BuildCacheControl()
	require.NotNil(t, cc)
}

func TestBuildCacheControlReturnsAnthropicType(t *testing.T) {
	opts := AnthropicOptions{CacheRetention: provider.CacheRetentionShort}
	cc := opts.BuildCacheControl()
	var _ *anthropic.CacheControlEphemeralParam = cc
}

func TestStealthToolRenameNoPrefix(t *testing.T) {
	assert.Equal(t, "computer_tool", StealthToolRename("computer"))
}

func TestStealthToolRenameAlreadyPrefixed(t *testing.T) {
	assert.Equal(t, "computer_tool", StealthToolRename("computer_tool"))
}

func TestStealthToolRenameEmpty(t *testing.T) {
	assert.Equal(t, "_tool", StealthToolRename(""))
}

func TestApplyStealthModeEmpty(t *testing.T) {
	tools := []provider.Tool{}
	result := ApplyStealthMode(tools)
	assert.Empty(t, result)
}

func TestApplyStealthModeRenamesTools(t *testing.T) {
	tools := []provider.Tool{
		{Name: "Read", Description: "read file"},
		{Name: "Write", Description: "write file"},
	}
	result := ApplyStealthMode(tools)
	assert.Equal(t, "Read_tool", result[0].Name)
	assert.Equal(t, "Write_tool", result[1].Name)
	assert.Equal(t, "read file", result[0].Description)
}

func TestApplyStealthModeSkipsAlreadySuffixed(t *testing.T) {
	tools := []provider.Tool{
		{Name: "Read_tool", Description: "read file"},
	}
	result := ApplyStealthMode(tools)
	assert.Equal(t, "Read_tool", result[0].Name)
}

func TestBuildAnthropicSystemPromptEmpty(t *testing.T) {
	opts := AnthropicOptions{}
	blocks := opts.BuildSystemPrompt("")
	assert.Empty(t, blocks)
}

func TestBuildAnthropicSystemPromptNoCache(t *testing.T) {
	opts := AnthropicOptions{CacheRetention: provider.CacheRetentionNone}
	blocks := opts.BuildSystemPrompt("You are helpful.")
	require.Len(t, blocks, 1)
	assert.Equal(t, "You are helpful.", blocks[0].Text)
}

func TestBuildAnthropicSystemPromptWithCache(t *testing.T) {
	opts := AnthropicOptions{CacheRetention: provider.CacheRetentionShort}
	blocks := opts.BuildSystemPrompt("You are helpful.")
	require.Len(t, blocks, 1)
	assert.Equal(t, "You are helpful.", blocks[0].Text)
	require.NotNil(t, blocks[0].CacheControl)
}
