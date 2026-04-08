package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ChatModel struct {
	viewport      viewport.Model
	theme         Theme
	messages      []chatMessage
	ready         bool
	width         int
	height        int
	autoScroll    bool
	renderedCache string
	renderedCount int
}

type chatMessage struct {
	role    string
	content string
}

func NewChatModel(theme Theme) ChatModel {
	return ChatModel{
		theme:      theme,
		autoScroll: true,
	}
}

func (m ChatModel) Init() tea.Cmd {
	return nil
}

func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		if !m.ready {
			m.viewport = viewport.New(
				viewport.WithWidth(msg.Width),
				viewport.WithHeight(3),
			)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(m.theme.Border)
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
		}
		m.invalidateCache()
		m.renderMessages()
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *ChatModel) SetHeight(height int) {
	m.height = height
	if m.ready {
		m.viewport.SetHeight(m.clampedHeight())
		m.renderMessages()
	}
}

func (m *ChatModel) SetWidth(w int) {
	m.width = w
	if m.ready {
		m.viewport.SetWidth(w)
		m.invalidateCache()
		m.renderMessages()
	}
}

func (m ChatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return m.viewport.View()
}

func (m *ChatModel) AddUserMessage(text string) {
	m.messages = append(m.messages, chatMessage{role: "user", content: text})
	m.invalidateCache()
	m.renderMessages()
}

func (m *ChatModel) AddAssistantMessage(text string) {
	m.messages = append(m.messages, chatMessage{role: "assistant", content: text})
	m.invalidateCache()
	m.renderMessages()
}

func (m *ChatModel) StartAssistantMessage() {
	m.messages = append(m.messages, chatMessage{role: "assistant", content: ""})
	m.invalidateCache()
	m.renderMessages()
}

func (m *ChatModel) AppendDelta(delta string) {
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].role != "assistant" {
		m.messages = append(m.messages, chatMessage{role: "assistant", content: ""})
		m.invalidateCache()
	}
	last := &m.messages[len(m.messages)-1]
	last.content += delta
	m.renderMessages()
}

func (m *ChatModel) AddErrorMessage(text string) {
	m.messages = append(m.messages, chatMessage{role: "error", content: text})
	m.invalidateCache()
	m.renderMessages()
}

func (m *ChatModel) RemoveTrailingEmptyAssistant() {
	if len(m.messages) == 0 {
		return
	}
	last := &m.messages[len(m.messages)-1]
	if last.role == "assistant" && strings.TrimSpace(last.content) == "" {
		m.messages = m.messages[:len(m.messages)-1]
		m.invalidateCache()
		m.renderMessages()
	}
}

func (m *ChatModel) Clear() {
	m.messages = []chatMessage{}
	m.invalidateCache()
	m.renderMessages()
}

func (m *ChatModel) invalidateCache() {
	m.renderedCache = ""
	m.renderedCount = 0
}

func (m ChatModel) clampedHeight() int {
	frameSize := 0
	if m.ready {
		frameSize = m.viewport.Style.GetVerticalFrameSize()
	}
	minHeight := frameSize + 1
	if m.height < minHeight {
		return minHeight
	}
	return m.height
}

func (m *ChatModel) renderMessages() {
	if !m.ready {
		return
	}

	contentWidth := m.width - m.viewport.Style.GetHorizontalFrameSize()
	if contentWidth < 10 {
		contentWidth = 10
	}

	var b strings.Builder

	if m.renderedCount > 0 && m.renderedCount <= len(m.messages) {
		b.WriteString(m.renderedCache)
		for _, msg := range m.messages[m.renderedCount:] {
			m.renderSingleMessage(&b, msg, contentWidth)
		}
	} else {
		for _, msg := range m.messages {
			m.renderSingleMessage(&b, msg, contentWidth)
		}
	}

	m.viewport.SetContent(b.String())
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

func (m *ChatModel) renderSingleMessage(b *strings.Builder, msg chatMessage, contentWidth int) {
	var prefix string
	switch msg.role {
	case "user":
		prefix = m.theme.UserStyle().Render("You: ")
	case "assistant":
		prefix = m.theme.MutedStyle().Render("Assistant: ")
	case "error":
		prefix = m.theme.ErrorStyle().Bold(true).Render("Error: ")
	}

	prefixWidth := lipgloss.Width(prefix)
	wrapWidth := contentWidth - prefixWidth
	if wrapWidth < 10 {
		wrapWidth = 10
	}

	wrapped := wrapLines(msg.content, wrapWidth)
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		if i == 0 {
			b.WriteString(prefix)
		} else {
			b.WriteString(strings.Repeat(" ", prefixWidth))
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func wrapLines(text string, width int) string {
	if width <= 0 {
		return text
	}
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line))
		b.WriteString("\n")
	}
	return b.String()
}
