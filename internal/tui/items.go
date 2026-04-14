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
	prefix := "  "
	if i.selected {
		prefix = "x "
	}
	return prefix + i.name
}

func (i knowledgeItem) Description() string {
	if i.selected {
		return "selected"
	}
	return "available"
}

func (i knowledgeItem) FilterValue() string { return i.name }
