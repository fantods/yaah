package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestPhase0Smoke(t *testing.T) {
	t.Log("tui package compiles")
}

func TestDefaultThemeColors(t *testing.T) {
	theme := DefaultTheme()

	assert.Equal(t, lipgloss.Color("#7D56F4"), theme.Primary)
	assert.Equal(t, lipgloss.Color("#313244"), theme.Secondary)
	assert.Equal(t, lipgloss.Color("#FF6E8A"), theme.Accent)
	assert.Equal(t, lipgloss.Color("#CDD6F4"), theme.Foreground)
	assert.Equal(t, lipgloss.Color("#6C7086"), theme.Muted)
	assert.Equal(t, lipgloss.Color("#45475A"), theme.Border)
	assert.Equal(t, lipgloss.Color("#F38BA8"), theme.Error)
	assert.Equal(t, lipgloss.Color("#A6E3A1"), theme.Success)
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
	assert.True(t, km.Help.Enabled())
	assert.True(t, km.Clear.Enabled())
	assert.True(t, km.ToggleThinking.Enabled())
}

func TestAgentEventMsgWrapping(t *testing.T) {
	theme := DefaultTheme()
	_ = NewChatModel(theme)
	_ = NewStreamingModel(theme)
	_ = NewThinkingModel(theme)
	_ = NewToolsModel(theme)
	_ = NewStatusModel(theme)
}

func TestStreamingModelAppendDelta(t *testing.T) {
	theme := DefaultTheme()
	m := NewStreamingModel(theme)

	m.AppendDelta("hello ")
	m.AppendDelta("world")

	assert.Equal(t, "hello world", m.Content())
}

func TestStreamingModelThinking(t *testing.T) {
	theme := DefaultTheme()
	m := NewStreamingModel(theme)

	m.SetThinking(true)
	m.AppendThinking("hmm")

	assert.True(t, m.IsThinking())
	assert.Equal(t, "hmm", m.Thinking())
	assert.Contains(t, m.View(), "Thinking:")
}

func TestStreamingModelReset(t *testing.T) {
	theme := DefaultTheme()
	m := NewStreamingModel(theme)

	m.AppendDelta("data")
	m.AppendThinking("think")
	m.SetThinking(true)
	m.Reset()

	assert.Equal(t, "", m.Content())
	assert.Equal(t, "", m.Thinking())
	assert.False(t, m.IsThinking())
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
