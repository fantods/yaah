package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fantods/yaah/internal/agent"
	"github.com/fantods/yaah/internal/logging"
	"github.com/fantods/yaah/internal/provider"
)

type appState int

const (
	stateIdle appState = iota
	stateStreaming
	stateModelPicker
)

type AppModel struct {
	theme     Theme
	keys      KeyMap
	chat      ChatModel
	input     InputModel
	thinking  ThinkingModel
	tools     ToolsModel
	status    StatusModel
	streaming StreamingModel
	picker    ModelPicker

	agent    *agent.Agent
	eventCh  <-chan agent.AgentEvent
	state    appState
	width    int
	height   int
	quitting bool
	showHelp bool
	lastErr  string
}

func NewAppModel(a *agent.Agent, initialModel provider.Model, catalog []provider.Model) AppModel {
	theme := DefaultTheme()
	keys := DefaultKeyMap()
	status := NewStatusModel(theme)
	status.SetModel(initialModel.Name)

	return AppModel{
		theme:     theme,
		keys:      keys,
		chat:      NewChatModel(theme),
		input:     NewInputModel(theme, keys),
		thinking:  NewThinkingModel(theme),
		tools:     NewToolsModel(theme),
		status:    status,
		streaming: NewStreamingModel(theme),
		picker:    NewModelPicker(theme, catalog),
		agent:     a,
		state:     stateIdle,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.input.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

		if m.state == stateModelPicker {
			return m.handlePickerKeys(msg)
		}

		switch msg.String() {
		case "enter":
			if m.input.Focused() && m.input.Value() != "" {
				text := m.input.Value()
				m.input.Reset()
				m.chat.AddUserMessage(text)
				m.state = stateStreaming
				m.status.SetStreaming(true)
				m.lastErr = ""

				ch := m.agent.Prompt(context.Background(), text)
				m.eventCh = ch
				cmds = append(cmds, waitForAgentEvents(ch))
				return m, tea.Batch(cmds...)
			}
		case "ctrl+x":
			if m.state == stateStreaming {
				m.agent.Abort()
				m.state = stateIdle
				m.status.SetStreaming(false)
			}
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "ctrl+l":
			m.chat.Clear()
			return m, nil
		case "ctrl+t":
			m.thinking.Toggle()
			return m, nil
		case "ctrl+m":
			if m.state == stateIdle {
				m.picker.Open(m.agent.ModelID())
				m.state = stateModelPicker
				m.input.Blur()
			}
			return m, nil
		}

	case agentEventMsg:
		cmd := m.handleAgentEvent(msg.Event)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case streamEndedMsg:
		if m.state == stateStreaming {
			m.state = stateIdle
			m.status.SetStreaming(false)
			m.lastErr = "stream ended unexpectedly"
		}
		m.eventCh = nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.chat, cmd = m.chat.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) handlePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.picker.CursorUp()
	case "down", "j":
		m.picker.CursorDown()
	case "enter":
		selected := m.picker.SelectedModel()
		m.agent.SetModel(selected)
		m.status.SetModel(selected.Name)
		m.picker.Close()
		m.state = stateIdle
		m.input.Focus()
		logging.Debug("model switched to %s (%s)", selected.Name, selected.ID)
		return m, nil
	case "esc", "ctrl+m":
		m.picker.Close()
		m.state = stateIdle
		m.input.Focus()
	}
	return m, nil
}

func (m *AppModel) handleAgentEvent(evt agent.AgentEvent) tea.Cmd {
	switch e := evt.(type) {
	case agent.AgentStartEvent:

	case agent.AgentEndEvent:
		m.state = stateIdle
		m.status.SetStreaming(false)
		m.thinking.Reset()
		m.tools.Reset()
		m.streaming.Reset()
		m.eventCh = nil
		if e.Error != nil {
			m.lastErr = e.Error.Error()
		}
		return nil

	case agent.TurnStartEvent:
		m.status.SetTurn(m.agent.State().GetTurn())

	case agent.TurnEndEvent:
		m.tools.Reset()
		m.thinking.Reset()

	case agent.MessageStartEvent:
		m.chat.StartAssistantMessage()

	case agent.MessageUpdateEvent:
		switch ev := e.AssistantMessageEvent.(type) {
		case provider.EventTextDelta:
			m.streaming.AppendDelta(ev.Delta)
			m.chat.AppendDelta(ev.Delta)
		}

	case agent.MessageEndEvent:

	case agent.ToolExecStartEvent:
		m.tools.AddTool(e.ToolName)

	case agent.ToolExecEndEvent:
		if e.IsError {
			m.tools.ErrorTool(e.ToolName)
		} else {
			m.tools.CompleteTool(e.ToolName)
		}
	}

	if m.eventCh != nil {
		return waitForAgentEvents(m.eventCh)
	}
	return nil
}

func (m AppModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	chatView := m.chat.View()
	inputView := m.input.View()
	toolsView := m.tools.View()
	thinkingView := m.thinking.View()
	statusView := m.status.View()

	var errLine string
	if m.lastErr != "" {
		errLine = m.theme.ErrorStyle().Render(fmt.Sprintf("  Error: %s", m.lastErr)) + "\n"
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		chatView,
		toolsView,
		thinkingView,
		inputView,
		errLine,
		statusView,
	)

	if m.state == stateModelPicker {
		pickerView := m.picker.View()
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, pickerView)
	} else if m.showHelp {
		helpText := m.renderHelp()
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpText)
	}

	return fmt.Sprintf("%s\n", body)
}

func (m AppModel) renderHelp() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(1, 2)

	lines := []string{
		"  Keybindings",
		"",
		"  enter         send message",
		"  shift+enter   newline",
		"  ctrl+c        quit",
		"  ctrl+x        abort streaming",
		"  ctrl+l        clear chat",
		"  ctrl+t        toggle thinking view",
		"  ctrl+m        switch model",
		"  ?             toggle this help",
		"",
		"  Press ? to close",
	}
	return style.Render(strings.Join(lines, "\n"))
}
