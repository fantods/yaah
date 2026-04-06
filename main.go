package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/fantods/yaah/internal/provider/anthropic"
	_ "github.com/fantods/yaah/internal/provider/openai"
	_ "github.com/fantods/yaah/internal/provider/zai"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fantods/yaah/internal/agent"
	"github.com/fantods/yaah/internal/logging"
	"github.com/fantods/yaah/internal/provider"
	"github.com/fantods/yaah/internal/tui"
)

func main() {
	debug := flag.String("debug", "", "enable debug logging to file (e.g. --debug yaah.log)")
	modelID := flag.String("model", "claude-sonnet-4-20250514", "model to use (run --list-models to see options)")
	listModels := flag.Bool("list-models", false, "list available models and exit")
	flag.Parse()

	if *listModels {
		fmt.Println("Available models:")
		for _, m := range provider.Catalog() {
			fmt.Printf("  %-30s %s (%s)\n", m.ID, m.Name, m.Provider)
		}
		return
	}

	if *debug != "" {
		logPath := *debug
		if !filepath.IsAbs(logPath) {
			abs, err := filepath.Abs(logPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error resolving log path: %v\n", err)
				os.Exit(1)
			}
			logPath = abs
		}

		if err := logging.Init(logPath); err != nil {
			fmt.Fprintf(os.Stderr, "error opening log file: %v\n", err)
			os.Exit(1)
		}
		defer logging.Close()

		logging.Debug("yaah started, debug log: %s", logPath)
	}

	model, err := provider.LookupModel(*modelID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	logging.Debug("model: %s, api: %s, maxTokens: %d", model.ID, model.API, model.MaxTokens)

	a := agent.NewAgent(
		agent.AgentOptions{
			Model: model,
			LoopConfig: agent.AgentLoopConfig{
				MaxTurns: 10,
			},
		},
		agent.StreamProxy,
	)

	p := tea.NewProgram(
		tui.NewAppModel(a, model, provider.Catalog()),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		logging.Debug("program error: %v", err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	logging.Debug("yaah exited normally")
}
