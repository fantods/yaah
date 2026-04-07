package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

type StatusModel struct {
	theme     Theme
	model     string
	turn      int
	streaming bool
	width     int
}

func NewStatusModel(theme Theme) StatusModel {
	return StatusModel{
		theme: theme,
	}
}

func (m *StatusModel) SetModel(name string) {
	m.model = name
}

func (m *StatusModel) SetTurn(turn int) {
	m.turn = turn
}

func (m *StatusModel) SetStreaming(v bool) {
	m.streaming = v
}

func (m *StatusModel) SetWidth(w int) {
	m.width = w
}

func (m StatusModel) View() string {
	w := m.width
	if w < 20 {
		w = 20
	}
	status := m.theme.StatusBarStyle().
		Width(w).
		Padding(0, 1).
		Render(fmt.Sprintf(" %s | Turn: %d | %s",
			m.model,
			m.turn,
			m.streamingIndicator(),
		))
	return status
}

func (m StatusModel) streamingIndicator() string {
	if m.streaming {
		return lipgloss.NewStyle().Foreground(m.theme.Accent).Render("streaming")
	}
	return lipgloss.NewStyle().Foreground(m.theme.Muted).Render("idle")
}
