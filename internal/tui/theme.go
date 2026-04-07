package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Primary    color.Color
	Secondary  color.Color
	Accent     color.Color
	Foreground color.Color
	Muted      color.Color
	Border     color.Color
	Error      color.Color
	Success    color.Color
}

func DefaultTheme() Theme {
	return Theme{
		Primary:    lipgloss.Color("#7D56F4"),
		Secondary:  lipgloss.Color("#313244"),
		Accent:     lipgloss.Color("#FF6E8A"),
		Foreground: lipgloss.Color("#CDD6F4"),
		Muted:      lipgloss.Color("#6C7086"),
		Border:     lipgloss.Color("#45475A"),
		Error:      lipgloss.Color("#F38BA8"),
		Success:    lipgloss.Color("#A6E3A1"),
	}
}

func (t Theme) UserStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)
}

func (t Theme) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Muted)
}

func (t Theme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Error)
}

func (t Theme) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderForeground(t.Border)
}

func (t Theme) StatusBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Foreground).
		Background(t.Secondary)
}

func (t Theme) ToolNameStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)
}

func (t Theme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Success)
}

func (t Theme) AcccentArrow() string {
	return lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Render(">")
}
