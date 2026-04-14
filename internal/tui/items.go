package tui

type homeItem struct {
	title       string
	description string
	action      string
}

func (i homeItem) Title() string       { return i.title }
func (i homeItem) Description() string { return i.description }
func (i homeItem) FilterValue() string { return i.title + " " + i.description }

type knowledgeItem struct {
	name     string
	selected bool
}

func (i knowledgeItem) Title() string {
	checkbox := "[ ] "
	style := checkboxStyle
	if i.selected {
		checkbox = "[x] "
		style = selectedCheckboxStyle
	}
	return style.Render(checkbox) + i.name
}

func (i knowledgeItem) Description() string {
	if i.selected {
		return "Enabled - will be context-loaded during prompt generation"
	}
	return "Available"
}

func (i knowledgeItem) FilterValue() string { return i.name }
