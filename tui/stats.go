package main

import (
	"fmt"
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
			m.scorecard = nil
			m.roundID = 0
			m.editingCell = false
			m.editBuf = ""
			m.state = statePlayerDetail
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

// ── Helpers ───────────────────────────────────────────────────────────────────

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
