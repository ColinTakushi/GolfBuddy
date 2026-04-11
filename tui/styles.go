package main

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorGreenDark  = lipgloss.Color("#1a5c1a")
	colorGreenMid   = lipgloss.Color("#4CAF50")
	colorGreenLight = lipgloss.Color("#81C784")
	colorGrayLight  = lipgloss.Color("#CCCCCC")
	colorGrayMid    = lipgloss.Color("#777777")
	colorGrayDark   = lipgloss.Color("#555555")
	colorWhite      = lipgloss.Color("#FFFFFF")
	colorBlack      = lipgloss.Color("#000000")
	colorRed        = lipgloss.Color("#F44336")
	colorRedDark    = lipgloss.Color("#cc1212")
	colorBlue       = lipgloss.Color("#3a32a8")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorGreenDark).
			PaddingLeft(2).
			PaddingRight(2)

	panelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorGreenMid).
			Padding(1, 2)

	menuPanelStyle   = panelStyle.MarginRight(1)
	infoPanelStyle   = panelStyle
	inputPanelStyle  = panelStyle
	outputPanelStyle = panelStyle

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorGreenMid).
				Bold(true)

	selectedSubStyle = lipgloss.NewStyle().
				Foreground(colorGreenLight).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(colorGrayLight)

	subItemStyle = lipgloss.NewStyle().
			Foreground(colorGrayMid)

	descStyle = lipgloss.NewStyle().
			Foreground(colorGrayLight)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorGrayMid)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(colorGreenLight).
			Bold(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(colorGreenMid).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreenMid).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorGrayDark).
			PaddingLeft(2).
			PaddingTop(1)

	// focusedCellStyle highlights the entire cell background when navigating
	focusedCellStyle = lipgloss.NewStyle().
				Foreground(colorBlack).
				Background(colorGreenMid).
				Bold(true)

	errorCellStyle = lipgloss.NewStyle().
			Foreground(colorBlack).
			Background(colorRedDark).
			Bold(true)

	editingCellStyle = lipgloss.NewStyle().
				Foreground(colorBlack).
				Background(colorBlue).
				Bold(true).
				Blink(true)
)
