package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type InputModel struct {
	ta    textarea.Model
	theme Theme
	keys  KeyMap
	width int
}

func NewInputModel(theme Theme, keys KeyMap) InputModel {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."

	styles := textarea.DefaultStyles(true)
	styles.Focused = textarea.StyleState{
		Base:        lipgloss.NewStyle(),
		Text:        lipgloss.NewStyle().Foreground(theme.Foreground),
		CursorLine:  lipgloss.NewStyle(),
		EndOfBuffer: lipgloss.NewStyle().Foreground(theme.Border),
		Placeholder: lipgloss.NewStyle().Foreground(theme.Muted),
		Prompt:      lipgloss.NewStyle().Foreground(theme.Primary),
	}
	styles.Blurred = textarea.StyleState{
		Base:        lipgloss.NewStyle(),
		Text:        lipgloss.NewStyle().Foreground(theme.Muted),
		CursorLine:  lipgloss.NewStyle(),
		EndOfBuffer: lipgloss.NewStyle().Foreground(theme.Border),
		Placeholder: lipgloss.NewStyle().Foreground(theme.Muted),
		Prompt:      lipgloss.NewStyle().Foreground(theme.Muted),
	}
	ta.SetStyles(styles)

	ta.Prompt = ""
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.ta.SetWidth(msg.Width - 4)
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m *InputModel) SetWidth(w int) {
	m.width = w
	m.ta.SetWidth(w - 4)
}

func (m InputModel) View() string {
	horizontal := m.theme.MutedStyle().Render(strings.Repeat("─", max(m.width, 20)))

	inner := m.ta.View()

	lines := strings.Split(inner, "\n")
	var b strings.Builder
	for i, line := range lines {
		b.WriteString("  ")
		trimmed := strings.TrimRight(line, " ")
		b.WriteString(trimmed)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}

	return horizontal + "\n" + b.String() + "\n" + horizontal
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
