package tui

import (
	"fmt"
	"strings"
)

type toolEntry struct {
	name   string
	status string
}

type ToolsModel struct {
	theme  Theme
	tools  []toolEntry
	active bool
}

func NewToolsModel(theme Theme) ToolsModel {
	return ToolsModel{
		theme: theme,
	}
}

func (m *ToolsModel) AddTool(name string) {
	m.tools = append(m.tools, toolEntry{name: name, status: "running"})
	m.active = true
}

func (m *ToolsModel) CompleteTool(name string) {
	for i := range m.tools {
		if m.tools[i].name == name {
			m.tools[i].status = "done"
			break
		}
	}
	allDone := true
	for _, t := range m.tools {
		if t.status != "done" {
			allDone = false
		}
	}
	if allDone {
		m.active = false
	}
}

func (m *ToolsModel) ErrorTool(name string) {
	for i := range m.tools {
		if m.tools[i].name == name {
			m.tools[i].status = "error"
			break
		}
	}
}

func (m *ToolsModel) Reset() {
	m.tools = []toolEntry{}
	m.active = false
}

func (m ToolsModel) View() string {
	if len(m.tools) == 0 {
		return ""
	}

	var b strings.Builder
	for _, t := range m.tools {
		switch t.status {
		case "running":
			b.WriteString(fmt.Sprintf("  %s %s", m.theme.ToolNameStyle().Render(t.name), m.theme.MutedStyle().Render("...")))
		case "done":
			b.WriteString(fmt.Sprintf("  %s %s", m.theme.ToolNameStyle().Render(t.name), m.theme.SuccessStyle().Render("done")))
		case "error":
			b.WriteString(fmt.Sprintf("  %s %s", m.theme.ToolNameStyle().Render(t.name), m.theme.ErrorStyle().Render("error")))
		}
		b.WriteString("\n")
	}
	return b.String()
}
