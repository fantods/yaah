package tui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/fantods/yaah/internal/message"
)

type agentPhase int

const (
	phaseIdle agentPhase = iota
	phaseStreaming
	phaseThinking
	phaseToolExec
)

type InputStatusModel struct {
	theme         Theme
	phase         agentPhase
	toolName      string
	model         string
	contextWindow int
	usage         message.Usage
	width         int
}

func NewInputStatusModel(theme Theme, modelName string, contextWindow int) InputStatusModel {
	return InputStatusModel{
		theme:         theme,
		model:         modelName,
		contextWindow: contextWindow,
	}
}

func (m *InputStatusModel) SetPhase(p agentPhase) {
	m.phase = p
}

func (m *InputStatusModel) SetToolName(name string) {
	m.toolName = name
}

func (m *InputStatusModel) SetPhaseIdle() {
	m.phase = phaseIdle
	m.toolName = ""
}

func (m *InputStatusModel) SetUsage(u message.Usage) {
	m.usage = u
}

func (m *InputStatusModel) SetWidth(w int) {
	m.width = w
}

func (m *InputStatusModel) SetModel(name string, contextWindow int) {
	m.model = name
	m.contextWindow = contextWindow
}

func (m InputStatusModel) View() string {
	w := m.width
	if w < 20 {
		w = 20
	}
	innerW := w - 2

	left := m.renderPhase()

	total := m.usage.Input + m.usage.Output + m.usage.CacheRead + m.usage.CacheWrite
	tokens := fmt.Sprintf(
		"in:%s  out:%s  cache:%s/%s",
		fmtTokens(m.usage.Input),
		fmtTokens(m.usage.Output),
		fmtTokens(m.usage.CacheRead),
		fmtTokens(m.usage.CacheWrite),
	)

	right := fmt.Sprintf("%s/%s", fmtTokens(total), fmtTokens(int64(m.contextWindow)))

	leftSide := fmt.Sprintf(" %s  %s  %s", left, m.model, tokens)
	rightSide := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(right)

	gap := innerW - lipgloss.Width(leftSide) - lipgloss.Width(rightSide)
	if gap < 1 {
		gap = 1
	}

	line := leftSide + lipgloss.NewStyle().Width(gap).Render("") + rightSide

	return m.theme.StatusBarStyle().
		Width(w).
		Padding(0, 1).
		Render(line)
}

func (m InputStatusModel) renderPhase() string {
	var dotColor color.Color
	var label string

	switch m.phase {
	case phaseIdle:
		dotColor = m.theme.Muted
		label = "idle"
	case phaseStreaming:
		dotColor = m.theme.Accent
		label = "streaming"
	case phaseThinking:
		dotColor = m.theme.Primary
		label = "thinking..."
	case phaseToolExec:
		dotColor = m.theme.Success
		label = "tool: " + m.toolName
	}

	dot := lipgloss.NewStyle().Foreground(dotColor).Render("●")
	return dot + " " + label
}

func fmtTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fm", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
