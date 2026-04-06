package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ChatModel struct {
	viewport   viewport.Model
	theme      Theme
	messages   []chatMessage
	ready      bool
	width      int
	height     int
	autoScroll bool
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
		m.height = msg.Height - 8
		if m.height < 1 {
			m.height = 1
		}
		if !m.ready {
			m.viewport = viewport.New(msg.Width, m.height)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(m.theme.Border)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = m.height
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ChatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return m.viewport.View()
}

func (m *ChatModel) AddUserMessage(text string) {
	m.messages = append(m.messages, chatMessage{role: "user", content: text})
	m.renderMessages()
}

func (m *ChatModel) AddAssistantMessage(text string) {
	m.messages = append(m.messages, chatMessage{role: "assistant", content: text})
	m.renderMessages()
}

func (m *ChatModel) StartAssistantMessage() {
	m.messages = append(m.messages, chatMessage{role: "assistant", content: ""})
	m.renderMessages()
}

func (m *ChatModel) AppendDelta(delta string) {
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].role != "assistant" {
		m.messages = append(m.messages, chatMessage{role: "assistant", content: ""})
	}
	last := &m.messages[len(m.messages)-1]
	last.content += delta
	m.renderMessages()
}

func (m *ChatModel) Clear() {
	m.messages = []chatMessage{}
	m.renderMessages()
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
	for _, msg := range m.messages {
		var prefix string
		switch msg.role {
		case "user":
			prefix = m.theme.UserStyle().Render("You: ")
		case "assistant":
			prefix = m.theme.MutedStyle().Render("Assistant: ")
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

	m.viewport.SetContent(b.String())
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
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
