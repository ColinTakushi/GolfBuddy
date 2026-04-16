package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Data types ────────────────────────────────────────────────────────────────

type playerEntry struct {
	Name     string
	Rounds   int
	PlayerID int
}

type roundEntry struct {
	ID     int
	Course string
	Date   string
	Score  int
	Par    int
	Diff   int
}

type playerStatsData struct {
	TotalRounds int
	AvgScore    float64
	BestScore   int
	WorstScore  int
	Handicap    float64
}

// ── Messages ──────────────────────────────────────────────────────────────────

type playerListMsg struct {
	players []playerEntry
	err     error
}

type roundListMsg struct {
	rounds []roundEntry
	stats  playerStatsData
	err    error
}

type roundDetailMsg struct {
	sc      *scorecardData
	roundID int
	err     error
}

type roundSavedMsg struct {
	err error
}

type roundDeletedMsg struct {
	err error
}

// ── Update handlers ───────────────────────────────────────────────────────────

func (m model) updatePlayerList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "backspace":
		m.state = stateMainMenu
	case "up", "k":
		if m.playerIdx > 0 {
			m.playerIdx--
		}
	case "down", "j":
		if m.playerIdx < len(m.players)-1 {
			m.playerIdx++
		}
	case "enter", "right", "l":
		if len(m.players) == 0 {
			break
		}
		m.playerName = m.players[m.playerIdx].Name
		m.rounds = nil
		m.state = statePlayerDetail
		m.playerId = m.players[m.playerIdx].PlayerID
		return m, cmdFetchRounds(m.playerName)
	}
	return m, nil
}

func (m model) updatePlayerDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "left", "h", "backspace":
		m.state = statePlayerList
	case "up", "k":
		if m.roundIdx > 0 {
			m.roundIdx--
		}
	case "down", "j":
		if m.roundIdx < len(m.rounds)-1 {
			m.roundIdx++
		}
	case "enter", "right", "l":
		if len(m.rounds) == 0 {
			break
		}
		re := m.rounds[m.roundIdx]
		m.scorecard = nil
		m.fromRoundsMenu = false
		m.state = stateRoundView
		return m, cmdFetchRoundDetail(m.playerName, re.ID)
	}
	return m, nil
}

func (m model) updateRoundView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Loading state
	if m.scorecard == nil {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	// Intercept esc, s, and d; delegate everything else to scorecard handlers
	if !m.editingCell {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "backspace":
			m.editingCell = false
			m.editBuf = ""
			if m.fromRoundsMenu {
				m.state = stateRoundSummaryView
			} else {
				m.scorecard = nil
				m.roundID = 0
				m.state = statePlayerDetail
			}
			return m, nil
		case "s":
			summary := formatRoundSummary(m.scorecard)
			m.output = summary
			sc := m.scorecard
			id := m.roundID
			m.scorecard = nil
			m.roundID = 0
			m.state = stateOutput
			return m, cmdUpdateRound(id, sc)
		}
	}

	// Delegate navigation and cell editing to existing scorecard logic
	return m.updateScorecard(msg)
}

func (m model) updateAllRoundsList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	groups := groupRounds(m.allRounds)

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc", "backspace":
		if m.allRoundPlayerIdx >= 0 {
			m.allRoundPlayerIdx = -1
			return m, nil
		}
		m.state = stateMainMenu

	case "up", "k":
		if m.allRoundPlayerIdx >= 0 {
			if m.allRoundPlayerIdx > 0 {
				m.allRoundPlayerIdx--
			}
		} else if m.allRoundIdx > 0 {
			m.allRoundIdx--
		}

	case "down", "j":
		if m.allRoundPlayerIdx >= 0 {
			group := groups[m.allRoundIdx]
			if m.allRoundPlayerIdx < len(group.Entries)-1 {
				m.allRoundPlayerIdx++
			}
		} else if m.allRoundIdx < len(groups)-1 {
			m.allRoundIdx++
		}

	case "enter", "right", "l":
		if len(groups) == 0 {
			break
		}
		group := groups[m.allRoundIdx]
		if m.allRoundPlayerIdx >= 0 {
			re := group.Entries[m.allRoundPlayerIdx]
			m.scorecard = nil
			m.fromRoundsMenu = true
			m.allRoundPlayerIdx = -1
			m.state = stateRoundSummaryView
			return m, cmdFetchRoundDetail(re.Player, re.ID)
		}
		if len(group.Entries) == 1 {
			re := group.Entries[0]
			m.scorecard = nil
			m.fromRoundsMenu = true
			m.state = stateRoundSummaryView
			return m, cmdFetchRoundDetail(re.Player, re.ID)
		}
		// Multiple players — enter player sub-selection
		m.allRoundPlayerIdx = 0
	}
	return m, nil
}

func (m model) updateRoundSummaryView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.scorecard == nil {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q", "esc", "backspace":
		m.scorecard = nil
		m.roundID = 0
		m.fromRoundsMenu = false
		m.state = stateAllRoundsList
	case "e":
		m.cursor = scCell{1, 0}
		m.editingCell = false
		m.state = stateRoundView
	}
	return m, nil
}

// ── Views ─────────────────────────────────────────────────────────────────────

func (m model) viewPlayerList() string {
	top    := "┌" + strings.Repeat("─", statsInner) + "┐"
	bottom := "└" + strings.Repeat("─", statsInner) + "┘"
	line   := func(s string) string { return "│" + fmt.Sprintf("%-*s", statsInner, s) + "│" }

	var sb strings.Builder
	sb.WriteString(top + "\n")

	title := centerPad(" SELECT PLAYER ", statsInner, '─')
	sb.WriteString("├" + title + "┤\n")
	sb.WriteString(line("") + "\n")

	if m.players == nil {
		sb.WriteString(line("  Loading players...") + "\n")
	} else if len(m.players) == 0 {
		sb.WriteString(line("  No players found.") + "\n")
	} else {
		for i, p := range m.players {
			rounds := fmt.Sprintf("%d round", p.Rounds)
			if p.Rounds != 1 {
				rounds += "s"
			}
			content := fmt.Sprintf("  %-*s %s", statsInner-12, p.Name, rounds)
			if i == m.playerIdx {
				content = selectedItemStyle.Render("> " + content[2:])
				sb.WriteString("│" + padRight(content, statsInner) + "│\n")
			} else {
				sb.WriteString(line(content) + "\n")
			}
		}
	}

	sb.WriteString(line("") + "\n")
	sb.WriteString(bottom)

	help := helpStyle.Render("↑/↓  navigate    enter  view stats    esc  back    q  quit")
	ui := lipgloss.JoinVertical(lipgloss.Left, sb.String(), help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

func (m model) viewPlayerDetail() string {
	top    := "┌" + strings.Repeat("─", statsInner) + "┐"
	mid    := "├" + strings.Repeat("─", statsInner) + "┤"
	bottom := "└" + strings.Repeat("─", statsInner) + "┘"
	line   := func(s string) string { return "│" + fmt.Sprintf("%-*s", statsInner, s) + "│" }

	var sb strings.Builder

	// Title bar
	title := centerPad(" "+strings.ToUpper(m.playerName)+" ", statsInner, '─')
	sb.WriteString("┌" + title + "┐\n")

	// Stats bar
	if m.rounds != nil {
		ps := m.playerStats
		statsLine := fmt.Sprintf("  Rounds: %d  │  Avg: %.1f  │  Best: %d  │  HCP: %.1f",
			ps.TotalRounds, ps.AvgScore, ps.BestScore, ps.Handicap)
		sb.WriteString(line(statsLine) + "\n")
	} else {
		sb.WriteString(line("  Loading...") + "\n")
	}
	sb.WriteString(mid + "\n")

	// Column header
	sb.WriteString(line(fmt.Sprintf("  %-*s %-*s %5s  %5s", statsDateCol, "DATE", statsCourseCol, "COURSE", "SCORE", "+/-")) + "\n")
	sb.WriteString(mid + "\n")

	if m.rounds == nil {
		sb.WriteString(line("  Loading rounds...") + "\n")
	} else if len(m.rounds) == 0 {
		sb.WriteString(line("  No rounds found.") + "\n")
	} else {
		for i, r := range m.rounds {
			sign := "+"
			if r.Diff < 0 {
				sign = ""
			}
			content := fmt.Sprintf("  %-*s %-*s %5d  %s%d", statsDateCol, r.Date, statsCourseCol, truncate(r.Course, statsCourseCol), r.Score, sign, r.Diff)
			if i == m.roundIdx {
				sb.WriteString("│" + padRight(selectedItemStyle.Render("> "+content[2:]), statsInner) + "│\n")
			} else {
				sb.WriteString(line(content) + "\n")
			}
		}
	}

	sb.WriteString(line("") + "\n")
	_ = top
	sb.WriteString(bottom)

	help := helpStyle.Render("↑/↓  navigate    enter  view round    esc  back   q  quit")
	ui := lipgloss.JoinVertical(lipgloss.Left, sb.String(), help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

func (m model) viewRoundView() string {
	if m.scorecard == nil {
		ui := dimStyle.Render("Loading round...")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
	}

	if m.width < minScorecardWidth {
		msg := fmt.Sprintf("Terminal must be at least %d chars wide (current: %d).", minScorecardWidth, m.width)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errorStyle.Render(msg))
	}

	table := m.renderScorecardTable()
	stats := formatRoundSummary(m.scorecard)

	var helpText string
	if m.editingCell {
		helpText = "enter/arrows  confirm    esc  cancel"
	} else {
		helpText = "↑/↓/←/→  navigate   e  edit   s  save   d  delete   esc  back"
	}
	help := helpStyle.Render(helpText)

	ui := lipgloss.JoinVertical(lipgloss.Left, table, "\n"+stats, help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

func (m model) viewRoundSummaryView() string {
	if m.scorecard == nil {
		ui := dimStyle.Render("Loading round...")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
	}
	summary := formatRoundSummary(m.scorecard)
	help := helpStyle.Render("e  edit    esc  back    q  quit")
	ui := lipgloss.JoinVertical(lipgloss.Left, summary, help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

func (m model) viewAllRoundsList() string {
	bottom   := "└" + strings.Repeat("─", statsInner) + "┘"
	line     := func(s string) string { return "│" + fmt.Sprintf("%-*s", statsInner, s) + "│" }
	// wrapCard places a 64-char card line inside the 68-char outer box.
	// Uses direct concatenation (not fmt.Sprintf padding) because box-drawing
	// characters are 3 bytes in UTF-8 but only 1 terminal column — byte-based
	// padding would misalign the borders.
	wrapCard := func(cardLine string) string { return "│  " + cardLine + "  │" }

	var sb strings.Builder
	title := centerPad(" ALL ROUNDS ", statsInner, '─')
	sb.WriteString("┌" + title + "┐\n")

	if m.allRounds == nil {
		sb.WriteString(line("") + "\n")
		sb.WriteString(line("  Loading rounds...") + "\n")
		sb.WriteString(line("") + "\n")
		sb.WriteString(bottom)
		help := helpStyle.Render("↑/↓  navigate    enter  select    esc  back    q  quit")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Left, sb.String(), help))
	}
	if len(m.allRounds) == 0 {
		sb.WriteString(line("") + "\n")
		sb.WriteString(line("  No rounds found.") + "\n")
		sb.WriteString(line("") + "\n")
		sb.WriteString(bottom)
		help := helpStyle.Render("↑/↓  navigate    enter  select    esc  back    q  quit")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Left, sb.String(), help))
	}

	groups := groupRounds(m.allRounds)
	for gi, group := range groups {
		sb.WriteString(line("") + "\n")

		// Card header — highlight with selectedItemStyle when this card is selected.
		// No padRight here: wrapCard concatenates strings directly, so the visual
		// width of the header is preserved regardless of ANSI escape codes.
		header := buildCardHeader(group.Date, group.Course, group.Entries[0].Par)
		if gi == m.allRoundIdx {
			header = selectedItemStyle.Render(header)
		}
		sb.WriteString(wrapCard(header) + "\n")

		// One row per player
		for pi, entry := range group.Entries {
			sign := "+"
			if entry.Diff < 0 {
				sign = ""
			}
			// Pure ASCII content — byte width equals visual width, no padRight needed.
			// Width breakdown: 2 + player(45) + 1 + score(5) + 2 + sign(1) + diff(4) + 2 = 62
			content := fmt.Sprintf("  %-*s %5d  %s%-4d  ",
				allRoundsCardInner-17, truncate(entry.Player, allRoundsCardInner-17),
				entry.Score, sign, entry.Diff)

			if gi == m.allRoundIdx && m.allRoundPlayerIdx == pi {
				styled := selectedItemStyle.Render("> " + content[2:])
				sb.WriteString(wrapCard("│" + padRight(styled, allRoundsCardInner) + "│") + "\n")
			} else {
				sb.WriteString(wrapCard("│" + content + "│") + "\n")
			}
		}

		sb.WriteString(wrapCard("└"+strings.Repeat("─", allRoundsCardInner)+"┘") + "\n")
	}

	sb.WriteString(line("") + "\n")
	sb.WriteString(bottom)

	helpText := "↑/↓  navigate    enter  select    esc  back    q  quit"
	if m.allRoundPlayerIdx >= 0 {
		helpText = "↑/↓  choose player    enter  view scorecard    esc  back"
	}
	help := helpStyle.Render(helpText)
	ui := lipgloss.JoinVertical(lipgloss.Left, sb.String(), help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// buildCardHeader builds the 64-char card top border line for one round group.
// Example: ┌─ 2026-04-11 · Emerald Ridge · Par 72 ─────────────────────────┐
func buildCardHeader(date, course string, par int) string {
	parStr    := strconv.Itoa(par)
	// Fixed overhead: "─ " + date + " · " + " · Par " + parStr + " ─"
	fixed     := 2 + len(date) + 3 + 7 + len(parStr) + 2
	maxCourse := allRoundsCardInner - fixed
	course     = truncate(course, maxCourse)
	titleText := date + " · " + course + " · Par " + parStr
	// Use lipgloss.Width, not len(), because "·" is a 2-byte UTF-8 char (U+00B7)
	// but only 1 terminal column wide — len() overcounts by 2.
	fill      := allRoundsCardInner - 4 - lipgloss.Width(titleText)
	if fill < 0 {
		fill = 0
	}
	inner := "─ " + titleText + " " + strings.Repeat("─", fill) + "─"
	return "┌" + inner + "┐"
}

// padRight pads s with spaces on the right to reach the given visual width.
// Uses lipgloss.Width so ANSI escape codes are not counted.
func padRight(s string, width int) string {
	pad := width - lipgloss.Width(s)
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

// centerPad centers text within width, padding with fill rune on both sides.
func centerPad(s string, width int, fill rune) string {
	sLen := len([]rune(s))
	if sLen >= width {
		return s[:width]
	}
	left := (width - sLen) / 2
	right := width - sLen - left
	return strings.Repeat(string(fill), left) + s + strings.Repeat(string(fill), right)
}

// truncate shortens a string to max length, adding "…" if cut.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
