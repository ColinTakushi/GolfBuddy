package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appState int

const (
	stateMainMenu appState = iota
	stateSubMenu
	stateInput
	stateOutput
	stateScorecard
	statePlayerList
	statePlayerDetail
	stateRoundView
)

const (
	leftWidth  = 28
	rightWidth = 99
	// Total rendered width: leftWidth + 7 (border+padding+margin) + rightWidth + 6 (border+padding) = 140
	titleContentWidth = leftWidth + rightWidth + 9  // = 136, renders to 140 with title padding
	bodyContentWidth  = leftWidth + rightWidth + 7  // = 134, renders to 140 with panel border+padding
)

// projectRoot is the GolfBuddy directory (parent of tui/).
var projectRoot string

func init() {
	wd, err := os.Getwd()
	if err != nil {
		projectRoot = "."
		return
	}
	if filepath.Base(wd) == "tui" {
		projectRoot = filepath.Dir(wd)
	} else {
		projectRoot = wd
	}
}

// ── Menu data ────────────────────────────────────────────────────────────────

type scanModeType int

const (
	scanModeNone  scanModeType = iota
	scanModeImage              // parse via Gemini, show scorecard editor
	scanModeJSON               // read JSON file directly, show scorecard editor
)

type subItem struct {
	label       string
	prompt      string
	interactive bool
	fileInput   bool         // enable tab file completion
	scanMode    scanModeType // non-zero = triggers scorecard editor instead of runSub
	cmd         []string     // "<input>" replaced with user value
}

type menuItem struct {
	label       string
	description string
	subItems    []subItem
}

var menu = []menuItem{
	{
		label:       "scan",
		description: "Scan a golf scorecard.\nSend an image to Gemini OCR\nor load from a pre-scanned JSON.",
		subItems: []subItem{
			{
				label:     "image",
				prompt:    "Enter path to scorecard image:",
				fileInput: true,
				scanMode:  scanModeImage,
			},
			{
				label:     "json",
				prompt:    "Enter path to JSON file:",
				fileInput: true,
				scanMode:  scanModeJSON,
			},
		},
	},
	{
		label:       "stats",
		description: "Browse player statistics.\nSelect a player to view\nrounds and scorecards.",
	},
	{
		label:       "nuke",
		description: "Delete ALL data and recreate\nthe database schema.\nThis cannot be undone.",
		subItems: []subItem{
			{
				label:  "confirm",
				prompt: `Type "yes" to confirm:`,
				cmd:    []string{"python3", "main.py", "nuke"},
			},
		},
	},
	{
		label:       "close",
		description: "Exit Golf Buddy.",
	},
}

// ── Messages ─────────────────────────────────────────────────────────────────

type cmdOutputMsg struct {
	output string
	err    error
}

type execDoneMsg struct {
	err error
}

// ── Model ─────────────────────────────────────────────────────────────────────

type model struct {
	state       appState
	menuIdx     int
	subIdx      int
	input       textinput.Model
	output      string
	width       int
	height      int
	completions []string
	compIdx     int
	// scorecard review (scan flow)
	scorecard   *scorecardData
	cursor      scCell
	editingCell bool
	editBuf     string
	// stats browser
	players     []playerEntry
	playerIdx   int
	playerName  string
	rounds      []roundEntry
	roundIdx    int
	playerStats playerStatsData
	roundID     int // DB scorecard ID for update; 0 = new
}

func initialModel() model {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = bodyContentWidth - 8
	return model{
		state:  stateMainMenu,
		input:  ti,
		width:  80,
		height: 24,
	}
}

func (m model) Init() tea.Cmd { return nil }

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case cmdParseScorecardMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Parse failed: "+msg.err.Error()) + "\n\n" + string(msg.output)
			m.state = stateOutput
			return m, nil
		}
		sc, err := parseScorecardJSON(msg.output)
		if err != nil {
			m.output = errorStyle.Render("Invalid JSON from parser: " + err.Error())
			m.state = stateOutput
			return m, nil
		}
		m.scorecard = sc
		m.cursor = scCell{1, 0}
		m.editingCell = false
		m.state = stateScorecard
		return m, nil

	case scorecardParsedMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Failed to read file: " + msg.err.Error())
			m.state = stateOutput
			return m, nil
		}
		m.scorecard = msg.data
		m.cursor = scCell{1, 0}
		m.editingCell = false
		m.state = stateScorecard
		return m, nil

	case scorecardSavedMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Save failed: "+msg.err.Error()) + "\n\n" + m.output
		}
		// on success, m.output already holds the formatted round summary
		m.state = stateOutput
		return m, nil

	case playerListMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Failed to load players: " + msg.err.Error())
			m.state = stateOutput
			return m, nil
		}
		m.players = msg.players
		m.playerIdx = 0
		return m, nil

	case roundListMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Failed to load rounds: " + msg.err.Error())
			m.state = stateOutput
			return m, nil
		}
		m.rounds = msg.rounds
		m.playerStats = msg.stats
		m.roundIdx = 0
		return m, nil

	case roundDetailMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Failed to load round: " + msg.err.Error())
			m.state = stateOutput
			return m, nil
		}
		m.scorecard = msg.sc
		m.roundID = msg.roundID
		m.cursor = scCell{1, 0}
		m.editingCell = false
		m.state = stateRoundView
		return m, nil

	case roundSavedMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Update failed: "+msg.err.Error()) + "\n\n" + m.output
		}
		m.state = stateOutput
		return m, nil

	case cmdOutputMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Error: "+msg.err.Error()) + "\n\n" + msg.output
		} else {
			m.output = msg.output
		}
		m.state = stateOutput
		return m, nil

	case execDoneMsg:
		if msg.err != nil {
			m.output = errorStyle.Render("Command failed: " + msg.err.Error())
			m.state = stateOutput
		} else {
			m.state = stateMainMenu
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateMainMenu:
			return m.updateMainMenu(msg)
		case stateSubMenu:
			return m.updateSubMenu(msg)
		case stateInput:
			return m.updateInput(msg)
		case stateScorecard:
			return m.updateScorecard(msg)
		case statePlayerList:
			return m.updatePlayerList(msg)
		case statePlayerDetail:
			return m.updatePlayerDetail(msg)
		case stateRoundView:
			return m.updateRoundView(msg)
		case stateOutput:
			m.state = stateMainMenu
			m.output = ""
			return m, nil
		}

	default:
		if m.state == stateInput ||
			((m.state == stateScorecard || m.state == stateRoundView) && m.editingCell && m.cursor.isNameCell()) {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.menuIdx > 0 {
			m.menuIdx--
		}
	case "down", "j":
		if m.menuIdx < len(menu)-1 {
			m.menuIdx++
		}
	case "enter", "right", "l":
		item := menu[m.menuIdx]
		if item.label == "close" {
			return m, tea.Quit
		}
		if item.label == "stats" {
			m.players = nil
			m.playerIdx = 0
			m.state = statePlayerList
			return m, cmdFetchPlayers()
		}
		m.subIdx = 0
		m.state = stateSubMenu
	}
	return m, nil
}

func (m model) updateSubMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	item := menu[m.menuIdx]
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "left", "h":
		m.state = stateMainMenu
	case "up", "k":
		if m.subIdx > 0 {
			m.subIdx--
		}
	case "down", "j":
		if m.subIdx < len(item.subItems)-1 {
			m.subIdx++
		}
	case "enter", "right", "l":
		sub := item.subItems[m.subIdx]
		if sub.prompt != "" {
			m.input.Reset()
			m.completions = nil
			m.state = stateInput
			return m, m.input.Focus()
		}
		return m.runSub(sub, "")
	}
	return m, nil
}

func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sub := menu[m.menuIdx].subItems[m.subIdx]

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.input.Blur()
		m.completions = nil
		m.state = stateSubMenu
		return m, nil

	case "tab":
		if sub.fileInput {
			m = m.cycleCompletions()
		}
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.input.Value())
		item := menu[m.menuIdx]
		if item.label == "nuke" && val != "yes" {
			m.output = errorStyle.Render(`Cancelled. You must type "yes" to confirm.`)
			m.input.Blur()
			m.completions = nil
			m.state = stateOutput
			return m, nil
		}
		m.input.Blur()
		m.completions = nil

		// Scan modes trigger the scorecard editor instead of a normal command
		switch sub.scanMode {
		case scanModeImage:
			return m, runParseImage(val)
		case scanModeJSON:
			return m, func() tea.Msg {
				data, err := readScorecardFile(val)
				return scorecardParsedMsg{data: data, err: err}
			}
		}
		return m.runSub(sub, val)

	default:
		// Any non-tab key resets completions
		m.completions = nil
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

// cycleCompletions advances or computes file path completions via glob.
func (m model) cycleCompletions() model {
	if len(m.completions) == 0 {
		pattern := filepath.Join(projectRoot, m.input.Value()+"*")
		matches, _ := filepath.Glob(pattern)
		for i, match := range matches {
			rel, err := filepath.Rel(projectRoot, match)
			if err == nil {
				matches[i] = rel
			}
		}
		m.completions = matches
		m.compIdx = 0
	} else {
		m.compIdx = (m.compIdx + 1) % len(m.completions)
	}
	if len(m.completions) > 0 {
		m.input.SetValue(m.completions[m.compIdx])
		m.input.CursorEnd()
	}
	return m
}

func (m model) runSub(sub subItem, input string) (tea.Model, tea.Cmd) {
	args := make([]string, len(sub.cmd))
	for i, a := range sub.cmd {
		if a == "<input>" {
			args[i] = input
		} else {
			args[i] = a
		}
	}

	name, cmdArgs := args[0], args[1:]

	if sub.interactive {
		c := exec.Command(name, cmdArgs...)
		c.Dir = projectRoot
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return execDoneMsg{err: err}
		})
	}

	return m, func() tea.Msg {
		var out bytes.Buffer
		c := exec.Command(name, cmdArgs...)
		c.Dir = projectRoot
		c.Stdout = &out
		c.Stderr = &out
		err := c.Run()
		return cmdOutputMsg{output: out.String(), err: err}
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	switch m.state {
	case stateMainMenu, stateSubMenu:
		ui := m.viewMenu()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
	case stateInput:
		ui := m.viewInput()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
	case stateOutput:
		ui := m.viewOutput()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
	case stateScorecard:
		return m.viewScorecard()
	case statePlayerList:
		return m.viewPlayerList()
	case statePlayerDetail:
		return m.viewPlayerDetail()
	case stateRoundView:
		return m.viewRoundView()
	}
	return ""
}

func (m model) viewMenu() string {
	title := titleStyle.Width(titleContentWidth).Render("WELCOME TO GOLF BUDDY")

	// Left panel
	var leftLines []string
	for i, item := range menu {
		if i == m.menuIdx {
			leftLines = append(leftLines, selectedItemStyle.Render("> "+item.label))
			if m.state == stateSubMenu {
				for j, sub := range item.subItems {
					if j == m.subIdx {
						leftLines = append(leftLines, selectedSubStyle.Render("  › "+sub.label))
					} else {
						leftLines = append(leftLines, subItemStyle.Render("    "+sub.label))
					}
				}
			}
		} else {
			leftLines = append(leftLines, normalItemStyle.Render("  "+item.label))
		}
	}
	leftPanel := menuPanelStyle.Width(leftWidth).Render(strings.Join(leftLines, "\n"))

	// Right panel
	selected := menu[m.menuIdx]
	var rightLines []string
	rightLines = append(rightLines, descStyle.Render(selected.description))
	if len(selected.subItems) > 0 {
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, dimStyle.Render("Options:"))
		for _, sub := range selected.subItems {
			rightLines = append(rightLines, dimStyle.Render("  "+sub.label))
		}
	}
	rightPanel := infoPanelStyle.Width(rightWidth).Render(strings.Join(rightLines, "\n"))

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	help := helpStyle.Render("↑/↓ navigate   enter select   esc back   q quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, panels, help)
}

func (m model) viewInput() string {
	item := menu[m.menuIdx]
	sub := item.subItems[m.subIdx]

	title := titleStyle.Width(titleContentWidth).Render("WELCOME TO GOLF BUDDY")

	content := breadcrumbStyle.Render(item.label+" › "+sub.label) +
		"\n\n" +
		promptStyle.Render(sub.prompt) +
		"\n" +
		m.input.View()

	// Show completion list when active
	if len(m.completions) > 0 {
		content += "\n\n" + dimStyle.Render("completions:")
		limit := len(m.completions)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			if i == m.compIdx {
				content += "\n" + selectedSubStyle.Render("› "+m.completions[i])
			} else {
				content += "\n" + dimStyle.Render("  "+m.completions[i])
			}
		}
		if len(m.completions) > 5 {
			content += "\n" + dimStyle.Render(fmt.Sprintf("  … and %d more", len(m.completions)-5))
		}
	}

	body := inputPanelStyle.Width(bodyContentWidth).Render(content)

	helpText := "enter confirm   esc back   ctrl+c quit"
	if sub.fileInput {
		helpText = "enter confirm   tab autocomplete   esc back   ctrl+c quit"
	}
	help := helpStyle.Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Left, title, body, help)
}

func (m model) viewOutput() string {
	title := titleStyle.Width(titleContentWidth).Render("WELCOME TO GOLF BUDDY")

	content := m.output
	if content == "" {
		content = successStyle.Render("Done!")
	}

	body := outputPanelStyle.Width(bodyContentWidth).Render(content)
	help := helpStyle.Render("any key to return")

	return lipgloss.JoinVertical(lipgloss.Left, title, body, help)
}
