package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type command struct {
	key   string
	label string
	desc  string
}

type CommandPalette struct {
	theme    Theme
	commands []command
	selected int
	active   bool
}

func NewCommandPalette(theme Theme) CommandPalette {
	return CommandPalette{
		theme: theme,
	}
}

func (cp *CommandPalette) Open(state appState) {
	cp.active = true
	cp.selected = 0
	cp.commands = commandsForState(state)
}

func (cp *CommandPalette) Close() {
	cp.active = false
}

func (cp *CommandPalette) IsActive() bool {
	return cp.active
}

func (cp *CommandPalette) CursorUp() {
	if cp.selected > 0 {
		cp.selected--
	}
}

func (cp *CommandPalette) CursorDown() {
	if cp.selected < len(cp.commands)-1 {
		cp.selected++
	}
}

func (cp *CommandPalette) SelectedCommand() string {
	if cp.selected < len(cp.commands) {
		return cp.commands[cp.selected].key
	}
	return ""
}

func (cp CommandPalette) View() string {
	if !cp.active || len(cp.commands) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(cp.theme.MutedStyle().Render("  Commands"))
	b.WriteString("\n\n")

	maxLabelLen := 0
	for _, cmd := range cp.commands {
		if len(cmd.label) > maxLabelLen {
			maxLabelLen = len(cmd.label)
		}
	}

	for i, cmd := range cp.commands {
		cursor := "  "
		label := cmd.label
		desc := cmd.desc

		if i == cp.selected {
			cursor = cp.theme.AcccentArrow()
			label = lipgloss.NewStyle().
				Foreground(cp.theme.Primary).
				Bold(true).
				Render(label)
			desc = lipgloss.NewStyle().
				Foreground(cp.theme.Accent).
				Render(desc)
		} else {
			label = cp.theme.MutedStyle().Render(label)
			desc = cp.theme.MutedStyle().Render(desc)
		}

		pad := strings.Repeat(" ", maxLabelLen-len(cmd.label)+1)
		b.WriteString(fmt.Sprintf("%s %s%s%s\n", cursor, label, pad, desc))
	}

	b.WriteString("\n")
	b.WriteString(cp.theme.MutedStyle().Render("  enter to run · esc to close"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cp.theme.Border).
		Padding(0, 2)

	return boxStyle.Render(b.String())
}

func commandsForState(state appState) []command {
	var cmds []command

	cmds = append(cmds, command{key: "switch-model", label: "Switch Model", desc: "change the active model"})

	switch state {
	case stateIdle:
		cmds = append(cmds,
			command{key: "clear", label: "Clear Chat", desc: "clear conversation history"},
			command{key: "toggle-thinking", label: "Toggle Thinking", desc: "show/hide thinking panel"},
		)
	case stateStreaming:
		cmds = append(cmds,
			command{key: "abort", label: "Abort Stream", desc: "stop current generation"},
			command{key: "toggle-thinking", label: "Toggle Thinking", desc: "show/hide thinking panel"},
		)
	}

	return cmds
}
