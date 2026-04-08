package tui

import (
	"image/color"
	"testing"

	"github.com/fantods/yaah/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhase0Smoke(t *testing.T) {
	t.Log("tui package compiles")
}

func TestDefaultThemeColors(t *testing.T) {
	theme := DefaultTheme()

	assert.Equal(t, color.RGBA{R: 0x7D, G: 0x56, B: 0xF4, A: 0xFF}, theme.Primary)
	assert.Equal(t, color.RGBA{R: 0x31, G: 0x32, B: 0x44, A: 0xFF}, theme.Secondary)
	assert.Equal(t, color.RGBA{R: 0xFF, G: 0x6E, B: 0x8A, A: 0xFF}, theme.Accent)
	assert.Equal(t, color.RGBA{R: 0xCD, G: 0xD6, B: 0xF4, A: 0xFF}, theme.Foreground)
	assert.Equal(t, color.RGBA{R: 0x6C, G: 0x70, B: 0x86, A: 0xFF}, theme.Muted)
	assert.Equal(t, color.RGBA{R: 0x45, G: 0x47, B: 0x5A, A: 0xFF}, theme.Border)
	assert.Equal(t, color.RGBA{R: 0xF3, G: 0x8B, B: 0xA8, A: 0xFF}, theme.Error)
	assert.Equal(t, color.RGBA{R: 0xA6, G: 0xE3, B: 0xA1, A: 0xFF}, theme.Success)
}

func TestThemeUserStyle(t *testing.T) {
	theme := DefaultTheme()
	s := theme.UserStyle()
	assert.True(t, s.GetBold())
	rendered := s.Render("hello")
	assert.Contains(t, rendered, "hello")
}

func TestThemeMutedStyle(t *testing.T) {
	theme := DefaultTheme()
	s := theme.MutedStyle()
	rendered := s.Render("dim")
	assert.Contains(t, rendered, "dim")
}

func TestThemeErrorStyle(t *testing.T) {
	theme := DefaultTheme()
	s := theme.ErrorStyle()
	rendered := s.Render("err")
	assert.Contains(t, rendered, "err")
}

func TestThemeStatusBarStyle(t *testing.T) {
	theme := DefaultTheme()
	s := theme.StatusBarStyle()
	rendered := s.Render("status")
	assert.Contains(t, rendered, "status")
}

func TestThemeToolNameStyle(t *testing.T) {
	theme := DefaultTheme()
	s := theme.ToolNameStyle()
	assert.True(t, s.GetBold())
}

func TestThemeSuccessStyle(t *testing.T) {
	theme := DefaultTheme()
	s := theme.SuccessStyle()
	rendered := s.Render("ok")
	assert.Contains(t, rendered, "ok")
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	assert.True(t, km.Up.Enabled())
	assert.True(t, km.Down.Enabled())
	assert.True(t, km.Enter.Enabled())
	assert.True(t, km.Quit.Enabled())
	assert.True(t, km.Newline.Enabled())
	assert.True(t, km.Abort.Enabled())
	assert.True(t, km.CommandPalette.Enabled())
	assert.True(t, km.Clear.Enabled())
	assert.True(t, km.ToggleThinking.Enabled())
	assert.True(t, km.SwitchModel.Enabled())
}

func TestAgentEventMsgWrapping(t *testing.T) {
	theme := DefaultTheme()
	_ = NewChatModel(theme)
	_ = NewThinkingModel(theme)
	_ = NewToolsModel(theme)
	_ = NewStatusModel(theme)
}

func TestThinkingModelToggle(t *testing.T) {
	theme := DefaultTheme()
	m := NewThinkingModel(theme)

	m.AppendContent("deep thoughts")
	m.Toggle()

	view := m.View()
	assert.Contains(t, view, "deep thoughts")
}

func TestThinkingModelCollapsed(t *testing.T) {
	theme := DefaultTheme()
	m := NewThinkingModel(theme)

	m.AppendContent("deep thoughts")

	view := m.View()
	assert.Contains(t, view, "Thinking...")
	assert.NotContains(t, view, "deep thoughts")
}

func TestThinkingModelReset(t *testing.T) {
	theme := DefaultTheme()
	m := NewThinkingModel(theme)

	m.AppendContent("x")
	m.Toggle()
	m.Reset()

	assert.Equal(t, "", m.View())
}

func TestToolsModelAddComplete(t *testing.T) {
	theme := DefaultTheme()
	m := NewToolsModel(theme)

	m.AddTool("search")
	m.CompleteTool("search")

	view := m.View()
	assert.Contains(t, view, "search")
	assert.Contains(t, view, "done")
}

func TestToolsModelError(t *testing.T) {
	theme := DefaultTheme()
	m := NewToolsModel(theme)

	m.AddTool("search")
	m.ErrorTool("search")

	view := m.View()
	assert.Contains(t, view, "error")
}

func TestToolsModelReset(t *testing.T) {
	theme := DefaultTheme()
	m := NewToolsModel(theme)

	m.AddTool("a")
	m.AddTool("b")
	m.Reset()

	assert.Equal(t, "", m.View())
}

func TestStatusModelView(t *testing.T) {
	theme := DefaultTheme()
	m := NewStatusModel(theme)

	m.SetModel("gpt-4")
	m.SetTurn(3)
	m.SetStreaming(true)

	view := m.View()
	assert.Contains(t, view, "gpt-4")
	assert.Contains(t, view, "Turn: 3")
	assert.Contains(t, view, "streaming")
}

func TestStatusModelIdle(t *testing.T) {
	theme := DefaultTheme()
	m := NewStatusModel(theme)

	m.SetModel("claude")
	m.SetStreaming(false)

	view := m.View()
	assert.Contains(t, view, "idle")
}

func TestChatModelAddMessages(t *testing.T) {
	theme := DefaultTheme()
	m := NewChatModel(theme)

	m.AddUserMessage("hello")
	m.AddAssistantMessage("hi there")

	assert.Equal(t, 2, len(m.messages))
}

func TestChatModelAppendDelta(t *testing.T) {
	theme := DefaultTheme()
	m := NewChatModel(theme)

	m.AppendDelta("hel")
	m.AppendDelta("lo")

	assert.Equal(t, 1, len(m.messages))
	assert.Equal(t, "hello", m.messages[0].content)
}

func TestFmtTokens(t *testing.T) {
	assert.Equal(t, "0", fmtTokens(0))
	assert.Equal(t, "832", fmtTokens(832))
	assert.Equal(t, "1.0k", fmtTokens(1000))
	assert.Equal(t, "8.2k", fmtTokens(8200))
	assert.Equal(t, "14.4k", fmtTokens(14400))
	assert.Equal(t, "1.0m", fmtTokens(1_000_000))
	assert.Equal(t, "2.5m", fmtTokens(2_500_000))
}

func TestInputStatusModelPhaseIdle(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)

	view := m.View()
	assert.Contains(t, view, "idle")
}

func TestInputStatusModelPhaseStreaming(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetPhase(phaseStreaming)

	view := m.View()
	assert.Contains(t, view, "streaming")
}

func TestInputStatusModelPhaseThinking(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetPhase(phaseThinking)

	view := m.View()
	assert.Contains(t, view, "thinking...")
}

func TestInputStatusModelPhaseToolExec(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetPhase(phaseToolExec)
	m.SetToolName("search")

	view := m.View()
	assert.Contains(t, view, "tool: search")
}

func TestInputStatusModelSetPhaseIdle(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetPhase(phaseToolExec)
	m.SetToolName("bash")
	m.SetPhaseIdle()

	view := m.View()
	assert.Contains(t, view, "idle")
	assert.NotContains(t, view, "bash")
}

func TestInputStatusModelUsage(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetUsage(message.Usage{Input: 8200, Output: 4100, CacheRead: 2100, CacheWrite: 0})

	view := m.View()
	assert.Contains(t, view, "8.2k")
	assert.Contains(t, view, "4.1k")
	assert.Contains(t, view, "2.1k")
	assert.Contains(t, view, "128.0k")
}

func TestInputStatusModelViewWidth(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(40)

	view := m.View()
	assert.Contains(t, view, "idle")
}

func TestInputStatusModelShowsModelName(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)

	view := m.View()
	assert.Contains(t, view, "gpt-4")
}

func TestInputStatusModelSetModel(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetModel("Claude Sonnet 4", 200000)

	view := m.View()
	assert.Contains(t, view, "Claude Sonnet 4")
	assert.Contains(t, view, "200.0k")
	assert.NotContains(t, view, "128.0k")
}

func TestInputStatusModelThinkingExpandedIdle(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetThinkingExpanded(true)

	view := m.View()
	assert.Contains(t, view, "thinking:on")
}

func TestInputStatusModelThinkingExpandedPhase(t *testing.T) {
	theme := DefaultTheme()
	m := NewInputStatusModel(theme, "gpt-4", 128000)
	m.SetWidth(80)
	m.SetThinkingExpanded(true)
	m.SetPhase(phaseThinking)

	view := m.View()
	assert.Contains(t, view, "thinking... [on]")
}

func TestCommandPaletteToggleThinkingSync(t *testing.T) {
	commands := commandsForState(stateIdle)
	var toggleCmd *command
	for i := range commands {
		if commands[i].key == "toggle-thinking" {
			toggleCmd = &commands[i]
			break
		}
	}
	require.NotNil(t, toggleCmd, "toggle-thinking should be in idle commands")
	assert.Equal(t, "toggle-thinking", toggleCmd.key)
}
