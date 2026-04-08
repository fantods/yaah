package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fantods/yaah/internal/agent"
	"github.com/fantods/yaah/internal/logging"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

type appState int

const (
	stateIdle appState = iota
	stateStreaming
	stateModelPicker
	stateCommandPalette
)

type AppModel struct {
	theme       Theme
	keys        KeyMap
	chat        ChatModel
	input       InputModel
	thinking    ThinkingModel
	tools       ToolsModel
	status      StatusModel
	picker      ModelPicker
	cmdPalette  CommandPalette
	inputStatus InputStatusModel

	agent    *agent.Agent
	eventCh  <-chan agent.AgentEvent
	state    appState
	width    int
	height   int
	quitting bool
	lastErr  string
}

func NewAppModel(a *agent.Agent, initialModel provider.Model, catalog []provider.Model) AppModel {
	theme := DefaultTheme()
	keys := DefaultKeyMap()
	status := NewStatusModel(theme)
	status.SetModel(initialModel.Name)

	return AppModel{
		theme:      theme,
		keys:       keys,
		chat:       NewChatModel(theme),
		input:      NewInputModel(theme, keys),
		thinking:   NewThinkingModel(theme),
		tools:      NewToolsModel(theme),
		status:     status,
		picker:     NewModelPicker(theme, catalog),
		cmdPalette: NewCommandPalette(theme),
		inputStatus: NewInputStatusModel(
			theme,
			initialModel.Name,
			initialModel.ContextWindow,
		),
		agent: a,
		state: stateIdle,
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
		m.input.SetWidth(msg.Width)
		m.status.SetWidth(msg.Width)
		m.inputStatus.SetWidth(msg.Width)
		m.chat.SetWidth(msg.Width)

	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

		m.lastErr = ""

		if m.state == stateModelPicker {
			return m.handlePickerKeys(msg)
		}
		if m.state == stateCommandPalette {
			return m.handlePaletteKeys(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Enter):
			if m.input.Focused() && m.input.Value() != "" {
				text := m.input.Value()
				m.input.Reset()
				m.chat.AddUserMessage(text)
				m.state = stateStreaming
				m.status.SetStreaming(true)

				ch := m.agent.Prompt(context.Background(), text)
				m.eventCh = ch
				cmds = append(cmds, waitForAgentEvents(ch))
				return m, tea.Batch(cmds...)
			}
		case key.Matches(msg, m.keys.Abort):
			if m.state == stateStreaming {
				m.agent.Abort()
				m.state = stateIdle
				m.status.SetStreaming(false)
			}
		case key.Matches(msg, m.keys.CommandPalette):
			m.cmdPalette.Open(m.state)
			m.state = stateCommandPalette
			m.input.Blur()
			return m, nil
		case key.Matches(msg, m.keys.Clear):
			m.chat.Clear()
			return m, nil
		case key.Matches(msg, m.keys.ToggleThinking):
			next := !m.agent.ThinkingEnabled()
			m.agent.SetThinkingEnabled(next)
			m.thinking.SetExpanded(next)
			m.inputStatus.SetThinkingExpanded(next)
			return m, nil
		case key.Matches(msg, m.keys.SwitchModel):
			if m.state == stateIdle {
				m.picker.Open(m.agent.ModelID())
				m.state = stateModelPicker
				m.input.Blur()
			}
		}

	case agentEventMsg:
		var cmd tea.Cmd
		m, cmd = m.handleAgentEvent(msg.Event)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case streamEndedMsg:
		if m.state == stateStreaming {
			m.state = stateIdle
			m.status.SetStreaming(false)
			m.chat.RemoveTrailingEmptyAssistant()
			m.chat.AddErrorMessage("Stream ended unexpectedly")
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

	m = m.computeLayout()

	return m, tea.Batch(cmds...)
}

func (m AppModel) handlePickerKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.picker.CursorUp()
	case key.Matches(msg, m.keys.Down):
		m.picker.CursorDown()
	case key.Matches(msg, m.keys.Enter):
		selected := m.picker.SelectedModel()
		m.agent.SetModel(selected)
		m.status.SetModel(selected.Name)
		m.inputStatus.SetModel(selected.Name, selected.ContextWindow)
		m.picker.Close()
		m.state = stateIdle
		m.input.Focus()
		logging.Debug("model switched to %s (%s)", selected.Name, selected.ID)
		return m, nil
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.SwitchModel):
		m.picker.Close()
		m.state = stateIdle
		m.input.Focus()
	}
	return m, nil
}

func (m AppModel) handleAgentEvent(evt agent.AgentEvent) (AppModel, tea.Cmd) {
	switch e := evt.(type) {
	case agent.AgentStartEvent:
		m.inputStatus.SetPhase(phaseGenerating)

	case agent.AgentEndEvent:
		m.state = stateIdle
		m.status.SetStreaming(false)
		m.thinking.Reset()
		m.tools.Reset()
		m.inputStatus.SetPhaseIdle()
		m.eventCh = nil
		if e.Error != nil {
			m.chat.RemoveTrailingEmptyAssistant()
			m.chat.AddErrorMessage(formatAPIError(e.Error))
			m.lastErr = e.Error.Error()
		}
		return m, nil

	case agent.TurnStartEvent:
		m.status.SetTurn(m.agent.State().GetTurn())
		m.inputStatus.SetPhase(phaseGenerating)

	case agent.TurnEndEvent:
		m.tools.Reset()
		m.thinking.Reset()

	case agent.MessageStartEvent:
		m.chat.StartAssistantMessage()

	case agent.MessageUpdateEvent:
		switch ev := e.AssistantMessageEvent.(type) {
		case provider.EventTextDelta:
			m.chat.AppendDelta(ev.Delta)
			m.inputStatus.SetPhase(phaseStreaming)
		case provider.EventThinkingStart:
			m.thinking.SetVisible(true)
			m.inputStatus.SetPhase(phaseThinking)
		case provider.EventThinkingDelta:
			m.thinking.AppendContent(ev.Delta)
			m.inputStatus.AppendThinking(ev.Delta)
			m.inputStatus.SetPhase(phaseThinking)
		case provider.EventThinkingEnd:
			m.thinking.SetVisible(true)
			m.inputStatus.SetPhase(phaseStreaming)
		}

	case agent.MessageEndEvent:
		if am, ok := e.Message.(message.AssistantMessage); ok {
			m.inputStatus.SetUsage(am.Usage)
		}

	case agent.ToolExecStartEvent:
		m.tools.AddTool(e.ToolName)
		m.inputStatus.SetPhase(phaseToolExec)
		m.inputStatus.SetToolName(e.ToolName)

	case agent.ToolExecEndEvent:
		if e.IsError {
			m.tools.ErrorTool(e.ToolName)
		} else {
			m.tools.CompleteTool(e.ToolName)
		}
		m.inputStatus.SetPhase(phaseStreaming)
	}

	if m.eventCh != nil {
		return m, waitForAgentEvents(m.eventCh)
	}
	return m, nil
}

func (m AppModel) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}

	chatView := m.chat.View()
	inputView := m.input.View()
	toolsView := m.tools.View()
	inputStatusView := m.inputStatus.View()
	statusView := m.status.View()

	var errLine string
	if m.lastErr != "" {
		errLine = m.theme.ErrorStyle().Render(fmt.Sprintf("  Error: %s", m.lastErr)) + "\n"
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		chatView,
		toolsView,
		inputStatusView,
		inputView,
		errLine,
		statusView,
	)

	if m.state == stateModelPicker {
		pickerView := m.picker.View()
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, pickerView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(m.theme.Secondary)),
		)
	} else if m.state == stateCommandPalette {
		paletteView := m.cmdPalette.View()
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, paletteView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(m.theme.Secondary)),
		)
	}

	v := tea.NewView(fmt.Sprintf("%s\n", body))
	v.AltScreen = true
	return v
}

func (m AppModel) computeLayout() AppModel {
	if m.width == 0 || m.height == 0 {
		return m
	}

	inputHeight := lipgloss.Height(m.input.View())
	statusHeight := lipgloss.Height(m.status.View())
	inputStatusHeight := lipgloss.Height(m.inputStatus.View())

	toolsHeight := 0
	if tv := m.tools.View(); tv != "" {
		toolsHeight = lipgloss.Height(tv)
	}

	errHeight := 0
	if m.lastErr != "" {
		errHeight = 1
	}

	chatHeight := m.height - inputHeight - statusHeight - toolsHeight - errHeight - inputStatusHeight
	m.chat.SetHeight(chatHeight)
	return m
}

func (m AppModel) handlePaletteKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.cmdPalette.CursorUp()
	case key.Matches(msg, m.keys.Down):
		m.cmdPalette.CursorDown()
	case key.Matches(msg, m.keys.Enter):
		cmd := m.cmdPalette.SelectedCommand()
		m.cmdPalette.Close()
		m.state = stateIdle
		m.input.Focus()

		switch cmd {
		case "switch-model":
			m.picker.Open(m.agent.ModelID())
			m.state = stateModelPicker
			m.input.Blur()
		case "clear":
			m.chat.Clear()
		case "toggle-thinking":
			next := !m.agent.ThinkingEnabled()
			m.agent.SetThinkingEnabled(next)
			m.thinking.SetExpanded(next)
			m.inputStatus.SetThinkingExpanded(next)
		case "abort":
			m.agent.Abort()
			m.state = stateIdle
			m.status.SetStreaming(false)
		}
		return m, nil
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.CommandPalette):
		m.cmdPalette.Close()
		m.state = stateIdle
		m.input.Focus()
	}
	return m, nil
}

func formatAPIError(err error) string {
	s := err.Error()

	if idx := strings.LastIndex(s, "{"); idx >= 0 {
		var body struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal([]byte(s[idx:]), &body) == nil {
			if body.Error.Message != "" {
				switch body.Error.Type {
				case "authentication_error":
					return "Authentication failed: " + body.Error.Message
				case "permission_error", "forbidden":
					return "Access denied: " + body.Error.Message
				case "rate_limit_error":
					return "Rate limited: " + body.Error.Message
				case "not_found_error":
					return "Model not found: " + body.Error.Message
				default:
					return body.Error.Message
				}
			}
			if body.Message != "" {
				return body.Message
			}
		}
	}

	if strings.Contains(s, "401") || strings.Contains(s, "authentication") {
		return "Authentication failed: API key is missing or invalid"
	}
	if strings.Contains(s, "403") {
		return "Access denied: insufficient permissions for this model"
	}
	if strings.Contains(s, "429") {
		return "Rate limited: too many requests, please wait and try again"
	}
	if strings.Contains(s, "500") || strings.Contains(s, "502") || strings.Contains(s, "503") {
		return "Provider is temporarily unavailable, please try again"
	}

	return s
}
