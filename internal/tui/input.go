package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputModel struct {
	ta    textarea.Model
	theme Theme
	keys  KeyMap
}

func NewInputModel(theme Theme, keys KeyMap) InputModel {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Muted)
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Muted)
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(theme.Primary)
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(theme.Muted)
	ta.Prompt = "> "
	ta.CharLimit = 0
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.Focus()

	ta.KeyMap.InsertNewline.SetKeys("shift+enter")

	return InputModel{
		ta:    ta,
		theme: theme,
		keys:  keys,
	}
}

func (m InputModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	return m.ta.View()
}

func (m *InputModel) Value() string {
	return m.ta.Value()
}

func (m *InputModel) Reset() {
	m.ta.Reset()
}

func (m *InputModel) Focus() {
	m.ta.Focus()
}

func (m *InputModel) Blur() {
	m.ta.Blur()
}

func (m InputModel) Focused() bool {
	return m.ta.Focused()
}
