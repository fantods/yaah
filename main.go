package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fantods/yaah/internal/agent"
	"github.com/fantods/yaah/internal/provider"
	"github.com/fantods/yaah/internal/tui"
)

func main() {
	model := provider.Model{
		ID:  "claude-sonnet-4-20250514",
		API: "anthropic-messages",
	}

	a := agent.NewAgent(
		agent.AgentOptions{
			Model: model,
			LoopConfig: agent.AgentLoopConfig{
				MaxTurns: 10,
			},
		},
		agent.StreamProxy,
	)

	p := tea.NewProgram(tui.NewAppModel(a), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
