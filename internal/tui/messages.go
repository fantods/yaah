package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fantods/yaah/internal/agent"
)

type agentEventMsg struct {
	Event agent.AgentEvent
}

type streamEndedMsg struct{}

func waitForAgentEvents(ch <-chan agent.AgentEvent) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return streamEndedMsg{}
		}
		return agentEventMsg{Event: evt}
	}
}
