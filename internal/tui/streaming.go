package tui

import "strings"

type StreamingModel struct {
	theme      Theme
	buffer     *strings.Builder
	thinking   string
	isThinking bool
}

func NewStreamingModel(theme Theme) StreamingModel {
	return StreamingModel{
		theme:  theme,
		buffer: &strings.Builder{},
	}
}

func (m *StreamingModel) AppendDelta(delta string) {
	m.buffer.WriteString(delta)
}

func (m *StreamingModel) AppendThinking(delta string) {
	m.thinking += delta
}

func (m *StreamingModel) SetThinking(v bool) {
	m.isThinking = v
}

func (m StreamingModel) Content() string {
	return m.buffer.String()
}

func (m StreamingModel) Thinking() string {
	return m.thinking
}

func (m StreamingModel) IsThinking() bool {
	return m.isThinking
}

func (m *StreamingModel) Reset() {
	m.buffer.Reset()
	m.thinking = ""
	m.isThinking = false
}

func (m StreamingModel) View() string {
	if m.isThinking && m.thinking != "" {
		return m.theme.MutedStyle().Render("Thinking: " + m.thinking)
	}
	return m.buffer.String()
}
