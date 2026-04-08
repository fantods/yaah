package tui

type ThinkingModel struct {
	theme    Theme
	content  string
	visible  bool
	expanded bool
}

func NewThinkingModel(theme Theme) ThinkingModel {
	return ThinkingModel{
		theme:   theme,
		visible: false,
	}
}

func (m ThinkingModel) Init() {
}

func (m *ThinkingModel) AppendContent(text string) {
	m.content += text
	m.visible = true
}

func (m *ThinkingModel) Toggle() {
	m.expanded = !m.expanded
}

func (m *ThinkingModel) SetExpanded(v bool) {
	m.expanded = v
}

func (m *ThinkingModel) SetVisible(v bool) {
	m.visible = v
}

func (m *ThinkingModel) Reset() {
	m.content = ""
	m.visible = false
}

func (m ThinkingModel) View() string {
	if !m.visible {
		return ""
	}
	if !m.expanded {
		return m.theme.MutedStyle().Render("  Thinking...")
	}
	return m.theme.MutedStyle().Render("  Thinking: " + m.content)
}
