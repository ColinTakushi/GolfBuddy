package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const apiBase = "http://localhost:8000"

// ── Data types ────────────────────────────────────────────────────────────────

type playerEntry struct {
	Name   string
	Rounds int
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

// ── HTTP fetch commands ───────────────────────────────────────────────────────

func cmdFetchPlayers() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(apiBase + "/users")
		if err != nil {
			return playerListMsg{err: err}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var raw []struct {
			Username       string `json:"username"`
			ScorecardsCount int   `json:"scorecards_count"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return playerListMsg{err: err}
		}
		players := make([]playerEntry, len(raw))
		for i, u := range raw {
			players[i] = playerEntry{Name: u.Username, Rounds: u.ScorecardsCount}
		}
		return playerListMsg{players: players}
	}
}

func cmdFetchRounds(name string) tea.Cmd {
	return func() tea.Msg {
		// Fetch round list
		resp, err := http.Get(apiBase + "/scorecards/" + name)
		if err != nil {
			return roundListMsg{err: err}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var rawRounds []struct {
			ID                int     `json:"id"`
			Course            string  `json:"course"`
			Date              string  `json:"date"`
			TotalScore        int     `json:"total_score"`
			TotalPar          int     `json:"total_par"`
			ScoreDifferential int     `json:"score_differential"`
		}
		if err := json.Unmarshal(body, &rawRounds); err != nil {
			return roundListMsg{err: err}
		}
		rounds := make([]roundEntry, len(rawRounds))
		for i, r := range rawRounds {
			date := r.Date
			if len(date) >= 10 {
				date = date[:10]
			}
			rounds[i] = roundEntry{
				ID:     r.ID,
				Course: r.Course,
				Date:   date,
				Score:  r.TotalScore,
				Par:    r.TotalPar,
				Diff:   r.ScoreDifferential,
			}
		}

		// Fetch aggregate stats
		var stats playerStatsData
		resp2, err := http.Get(apiBase + "/stats/" + name)
		if err == nil {
			defer resp2.Body.Close()
			body2, _ := io.ReadAll(resp2.Body)
			var rawStats struct {
				TotalRounds      int     `json:"total_rounds"`
				AverageScore     float64 `json:"average_score"`
				BestScore        int     `json:"best_score"`
				WorstScore       int     `json:"worst_score"`
				HandicapEstimate float64 `json:"handicap_estimate"`
			}
			if json.Unmarshal(body2, &rawStats) == nil {
				stats = playerStatsData{
					TotalRounds: rawStats.TotalRounds,
					AvgScore:    rawStats.AverageScore,
					BestScore:   rawStats.BestScore,
					WorstScore:  rawStats.WorstScore,
					Handicap:    rawStats.HandicapEstimate,
				}
			}
		}

		return roundListMsg{rounds: rounds, stats: stats}
	}
}

func cmdFetchRoundDetail(name string, id int) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("%s/scorecards/%s/%d", apiBase, name, id)
		resp, err := http.Get(url)
		if err != nil {
			return roundDetailMsg{err: err}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var raw struct {
			ID     int    `json:"id"`
			User   string `json:"user"`
			Course string `json:"course"`
			Holes  []struct {
				HoleNumber int `json:"hole_number"`
				Score      int `json:"score"`
				Par        int `json:"par"`
			} `json:"holes"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return roundDetailMsg{err: err}
		}
		if len(raw.Holes) != 18 {
			return roundDetailMsg{err: fmt.Errorf("expected 18 holes, got %d", len(raw.Holes))}
		}

		sc := &scorecardData{CourseName: raw.Course}
		pd := playerData{Name: raw.User}
		for _, h := range raw.Holes {
			i := h.HoleNumber - 1
			if i >= 0 && i < 18 {
				sc.HolePars[i] = h.Par
				pd.Scores[i] = h.Score
			}
		}
		sc.Players = []playerData{pd}

		return roundDetailMsg{sc: sc, roundID: raw.ID}
	}
}

func cmdNukeDatabase() tea.Cmd {
	return func() tea.Msg {
		req, _ := http.NewRequest(http.MethodDelete, apiBase+"/nuke", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return cmdOutputMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return cmdOutputMsg{err: fmt.Errorf("API error %d: %s", resp.StatusCode, body)}
		}
		return cmdOutputMsg{output: "Database cleared successfully."}
	}
}

func cmdUpdateRound(roundID int, sc *scorecardData) tea.Cmd {
	return func() tea.Msg {
		scores := sc.Players[0].Scores[:]
		body, _ := json.Marshal(scores)
		url := fmt.Sprintf("%s/scorecards/%d", apiBase, roundID)
		req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return roundSavedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			b, _ := io.ReadAll(resp.Body)
			return roundSavedMsg{err: fmt.Errorf("API error %d: %s", resp.StatusCode, b)}
		}
		return roundSavedMsg{}
	}
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

	// Intercept esc and s; delegate everything else to scorecard handlers
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

const statsPanelWidth = 70

func (m model) viewPlayerList() string {
	const inner = statsPanelWidth - 2

	top    := "┌" + strings.Repeat("─", inner) + "┐"
	bottom := "└" + strings.Repeat("─", inner) + "┘"
	line   := func(s string) string { return "│" + fmt.Sprintf("%-*s", inner, s) + "│" }

	var sb strings.Builder
	sb.WriteString(top + "\n")

	title := centerPad(" SELECT PLAYER ", inner, '─')
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
			content := fmt.Sprintf("  %-*s %s", inner-12, p.Name, rounds)
			if i == m.playerIdx {
				content = selectedItemStyle.Render("> " + content[2:])
				sb.WriteString("│" + fmt.Sprintf("%-*s", inner, content) + "│\n")
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
	const inner = statsPanelWidth - 2

	top    := "┌" + strings.Repeat("─", inner) + "┐"
	mid    := "├" + strings.Repeat("─", inner) + "┤"
	bottom := "└" + strings.Repeat("─", inner) + "┘"
	line   := func(s string) string { return "│" + fmt.Sprintf("%-*s", inner, s) + "│" }

	var sb strings.Builder

	// Title bar
	title := centerPad(" "+strings.ToUpper(m.playerName)+" ", inner, '─')
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
	sb.WriteString(line(fmt.Sprintf("  %-12s %-28s %5s  %5s", "DATE", "COURSE", "SCORE", "+/-")) + "\n")
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
			content := fmt.Sprintf("  %-12s %-28s %5d  %s%d", r.Date, truncate(r.Course, 28), r.Score, sign, r.Diff)
			if i == m.roundIdx {
				sb.WriteString("│" + fmt.Sprintf("%-*s", inner, selectedItemStyle.Render("> "+content[2:])) + "│\n")
			} else {
				sb.WriteString(line(content) + "\n")
			}
		}
	}

	sb.WriteString(line("") + "\n")
	_ = top
	sb.WriteString(bottom)

	help := helpStyle.Render("↑/↓  navigate    enter  view round    esc  back")
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
		helpText = "↑/↓/←/→  navigate    e  edit    s  save    esc  back"
	}
	help := helpStyle.Render(helpText)

	ui := lipgloss.JoinVertical(lipgloss.Left, table, "\n"+stats, help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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
