package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fantods/yaah/internal/agent"
)

type agentEventMsg struct {
	Event agent.AgentEvent
}

func waitForAgentEvents(ch <-chan agent.AgentEvent) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return nil
		}
		return agentEventMsg{Event: evt}
	}
}
