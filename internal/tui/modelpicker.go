package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/fantods/yaah/internal/provider"
)

type ModelPicker struct {
	theme    Theme
	models   []provider.Model
	selected int
	active   bool
}

func NewModelPicker(theme Theme, models []provider.Model) ModelPicker {
	return ModelPicker{
		theme:  theme,
		models: models,
	}
}

func (m *ModelPicker) Open(currentModelID string) {
	m.active = true
	for i, model := range m.models {
		if model.ID == currentModelID {
			m.selected = i
			return
		}
	}
	m.selected = 0
}

func (m *ModelPicker) Close() {
	m.active = false
}

func (m *ModelPicker) IsActive() bool {
	return m.active
}

func (m *ModelPicker) CursorUp() {
	if m.selected > 0 {
		m.selected--
	}
}

func (m *ModelPicker) CursorDown() {
	if m.selected < len(m.models)-1 {
		m.selected++
	}
}

func (m *ModelPicker) SelectedModel() provider.Model {
	return m.models[m.selected]
}

func (m ModelPicker) View() string {
	if !m.active || len(m.models) == 0 {
		return ""
	}

	maxNameLen := 0
	for _, model := range m.models {
		if len(model.Name) > maxNameLen {
			maxNameLen = len(model.Name)
		}
	}

	var b strings.Builder
	b.WriteString(m.theme.MutedStyle().Render("  Select a model:"))
	b.WriteString("\n\n")

	for i, model := range m.models {
		cursor := "  "
		name := model.Name
		info := fmt.Sprintf("(%s)", model.Provider)

		if i == m.selected {
			cursor = m.theme.AccentArrow()
			name = lipgloss.NewStyle().
				Foreground(m.theme.Primary).
				Bold(true).
				Render(name)
			info = lipgloss.NewStyle().
				Foreground(m.theme.Accent).
				Render(info)
		} else {
			name = m.theme.MutedStyle().Render(name)
			info = m.theme.MutedStyle().Render(info)
		}

		pad := strings.Repeat(" ", maxNameLen-len(m.models[i].Name)+1)
		b.WriteString(fmt.Sprintf("%s %s%s%s\n", cursor, name, pad, info))
	}

	b.WriteString("\n")
	b.WriteString(m.theme.MutedStyle().Render("  enter to select · esc to cancel"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 2)

	return boxStyle.Render(b.String())
}
