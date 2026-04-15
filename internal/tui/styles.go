package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor  = lipgloss.Color("81")  // Cyan/Blue
	accentColor   = lipgloss.Color("205") // Pink/Magenta
	successColor  = lipgloss.Color("78")  // Green
	warningColor  = lipgloss.Color("214") // Orange
	errorColor    = lipgloss.Color("203") // Red
	inactiveColor = lipgloss.Color("244") // Gray
	textColor     = lipgloss.Color("255") // White
	headerBgColor = lipgloss.Color("62")  // Purple-ish

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(headerBgColor).
			Padding(0, 2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")).
			Italic(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Margin(0, 1)

	accentStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	highlightStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	noticeStyle = lipgloss.NewStyle().
			Foreground(successColor)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(inactiveColor)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("45")).
			Bold(true)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	selectedCheckboxStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	// Result Page Styles
	promptPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(accentColor).
				Padding(1, 2)

	techInfoStyle = lipgloss.NewStyle().
			Foreground(inactiveColor).
			Italic(true)
)
