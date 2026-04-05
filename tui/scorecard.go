package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Data types ────────────────────────────────────────────────────────────────

type scorecardData struct {
	CourseName string       `json:"courseName"`
	HolePars   [18]int      `json:"holePars"`
	Players    []playerData `json:"players"`
	ImagePath  string       `json:"imagePath"`
}

type playerData struct {
	Name   string  `json:"name"`
	Scores [18]int `json:"scores"`
}

// rawScorecardJSON mirrors the Gemini/pipeline JSON structure for parsing.
type rawScorecardJSON struct {
	Course struct {
		Name     string `json:"name"`
		HolePars []int  `json:"holePars"`
	} `json:"course"`
	Players []struct {
		Name   string `json:"name"`
		Scores []int  `json:"scores"`
	} `json:"players"`
	ImagePath string `json:"imagePath"`
}

func parseScorecardJSON(raw []byte) (*scorecardData, error) {
	var r rawScorecardJSON
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	sc := &scorecardData{
		CourseName: r.Course.Name,
		ImagePath:  r.ImagePath,
	}
	for i := 0; i < 18 && i < len(r.Course.HolePars); i++ {
		sc.HolePars[i] = r.Course.HolePars[i]
	}
	for _, p := range r.Players {
		pd := playerData{Name: p.Name}
		for i := 0; i < 18 && i < len(p.Scores); i++ {
			pd.Scores[i] = p.Scores[i]
		}
		sc.Players = append(sc.Players, pd)
	}
	return sc, nil
}

func readScorecardFile(path string) (*scorecardData, error) {
	full := filepath.Join(projectRoot, path)
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	return parseScorecardJSON(raw)
}

// ── Messages ──────────────────────────────────────────────────────────────────

type scorecardParsedMsg struct {
	data *scorecardData
	err  error
}

// cmdParseScorecardMsg is returned after a non-interactive parse command finishes.
type cmdParseScorecardMsg struct {
	output []byte
	err    error
}

// scorecardSavedMsg is returned after the save command completes.
// On success we don't use the Python stdout — the summary was already formatted in Go.
type scorecardSavedMsg struct{ err error }

func runParseImage(imagePath string) tea.Cmd {
	return func() tea.Msg {
		var out bytes.Buffer
		c := exec.Command("python3", "scan.py", "image", imagePath, "--parse")
		c.Dir = projectRoot
		c.Stdout = &out
		c.Stderr = &out
		err := c.Run()
		return cmdParseScorecardMsg{output: out.Bytes(), err: err}
	}
}

// ── Cursor ────────────────────────────────────────────────────────────────────

// scCell identifies an editable cell.
// row: -1 = course name, 0 = par row, 1..N = player rows
// col: -1 = name column, 0..17 = holes 1–18
type scCell struct{ row, col int }

func (c scCell) isNameCell() bool { return c.col == -1 }
func (c scCell) isNumberCell() bool {
	return c.col >= 0 && c.col < 18
}

func (c scCell) maxRow(sc *scorecardData) int {
	return len(sc.Players) // row 0 = par, 1..N = players
}

func moveCursor(cur scCell, dr, dc int, sc *scorecardData) scCell {
	maxR := len(sc.Players)
	newCol := cur.col + dc
	newRow := cur.row + dr

	// col bounds: -1 (name) to 17 (hole 18) — clamp, no wrap
	if newCol < -1 {
		newCol = -1
	} else if newCol > 17 {
		newCol = 17
	}

	// row bounds: clamp to 0..maxR normally; name col also allows -1
	minRow := 0
	if newCol == -1 {
		minRow = -1 // course name row reachable on name column
	}
	if newRow < minRow {
		newRow = minRow
	} else if newRow > maxR {
		newRow = maxR
	}

	return scCell{newRow, newCol}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) updateScorecard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editingCell {
		return m.updateScorecardEdit(msg)
	}
	return m.updateScorecardNav(msg)
}

func (m model) updateScorecardNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q", "esc":
		m.scorecard = nil
		m.state = stateMainMenu
	case "up", "k":
		m.cursor = moveCursor(m.cursor, -1, 0, m.scorecard)
	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, 0, m.scorecard)
	case "left", "h":
		m.cursor = moveCursor(m.cursor, 0, -1, m.scorecard)
	case "right", "l":
		m.cursor = moveCursor(m.cursor, 0, 1, m.scorecard)
	case "enter", "e":
		m.editingCell = true
		m.editBuf = ""
		if m.cursor.isNameCell() {
			// pre-fill textinput with current name
			var name string
			if m.cursor.row == -1 {
				name = m.scorecard.CourseName
			} else {
				name = m.scorecard.Players[m.cursor.row-1].Name
			}
			m.input.SetValue(name)
			m.input.CursorEnd()
			return m, m.input.Focus()
		}
	case "s":
		return m.saveScorecard()
	}
	return m, nil
}

func (m model) updateScorecardEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Name cell edit — delegate to textinput
	if m.cursor.isNameCell() {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.editingCell = false
			m.input.Blur()
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name != "" {
				if m.cursor.row == -1 {
					m.scorecard.CourseName = name
				} else {
					m.scorecard.Players[m.cursor.row-1].Name = name
				}
			}
			m.editingCell = false
			m.input.Blur()
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Number cell edit
	key := msg.String()
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.editingCell = false
		m.editBuf = ""
	case "enter", "up", "down", "left", "right":
		m.editingCell = false
		m.editBuf = ""
		if key != "enter" {
			dr, dc := 0, 0
			switch key {
			case "up":
				dr = -1
			case "down":
				dr = 1
			case "left":
				dc = -1
			case "right":
				dc = 1
			}
			m.cursor = moveCursor(m.cursor, dr, dc, m.scorecard)
		}
	case "backspace", "delete":
		if len(m.editBuf) > 0 {
			m.editBuf = m.editBuf[:len(m.editBuf)-1]
		}
		if m.editBuf == "" {
			m.setScorecardCell(m.cursor, 0)
		} else if val, err := strconv.Atoi(m.editBuf); err == nil {
			m.setScorecardCell(m.cursor, val)
		}
	default:
		if len(key) == 1 && unicode.IsDigit(rune(key[0])) {
			m.editBuf += key
			if len(m.editBuf) > 2 {
				m.editBuf = m.editBuf[len(m.editBuf)-2:]
			}
			if val, err := strconv.Atoi(m.editBuf); err == nil && val > 0 {
				m.setScorecardCell(m.cursor, val)
			}
		}
	}
	return m, nil
}

func (m *model) setScorecardCell(c scCell, val int) {
	if c.row == 0 {
		m.scorecard.HolePars[c.col] = val
	} else if c.row > 0 && c.row-1 < len(m.scorecard.Players) {
		m.scorecard.Players[c.row-1].Scores[c.col] = val
	}
}

// formatRoundSummary builds box-drawing output for every player in the scorecard.
func formatRoundSummary(sc *scorecardData) string {
	const w = 66 // inner width between │ characters

	top    := "┌" + strings.Repeat("─", w) + "┐"
	mid    := "├" + strings.Repeat("─", w) + "┤"
	bottom := "└" + strings.Repeat("─", w) + "┘"

	line := func(s string) string {
		return fmt.Sprintf("│ %-*s │", w-2, s)
	}

	parTotal := 0
	parFront := 0
	parBack  := 0
	for i, p := range sc.HolePars {
		parTotal += p
		if i < 9 {
			parFront += p
		} else {
			parBack += p
		}
	}

	var parts []string
	for _, p := range sc.Players {
		scoreTotal := 0
		scoreFront := 0
		scoreBack  := 0
		birdies, pars, bogeys, doubles := 0, 0, 0, 0

		for i, s := range p.Scores {
			scoreTotal += s
			if i < 9 {
				scoreFront += s
			} else {
				scoreBack += s
			}
			diff := s - sc.HolePars[i]
			switch {
			case diff < 0:
				birdies++
			case diff == 0:
				pars++
			case diff == 1:
				bogeys++
			default:
				doubles++
			}
		}

		diffTotal := scoreTotal - parTotal
		diffFront := scoreFront - parFront
		diffBack  := scoreBack  - parBack

		sign := func(d int) string {
			if d >= 0 {
				return fmt.Sprintf("+%d", d)
			}
			return strconv.Itoa(d)
		}

		var sb strings.Builder
		sb.WriteString(top + "\n")
		sb.WriteString(line(fmt.Sprintf("PLAYER: %s", p.Name)) + "\n")
		sb.WriteString(mid + "\n")
		sb.WriteString(line(fmt.Sprintf("COURSE: %-40s PAR: %d", sc.CourseName, parTotal)) + "\n")
		sb.WriteString(mid + "\n")
		sb.WriteString(line(fmt.Sprintf(" TOTAL: %d  (vs. %d par)  [%s]", scoreTotal, parTotal, sign(diffTotal))) + "\n")
		sb.WriteString(line(fmt.Sprintf(" FRONT: %d  (vs. %d par)  [%s]", scoreFront, parFront, sign(diffFront))) + "\n")
		sb.WriteString(line(fmt.Sprintf(" BACK:  %d  (vs. %d par)  [%s]", scoreBack, parBack, sign(diffBack))) + "\n")
		sb.WriteString(mid + "\n")
		sb.WriteString(line(fmt.Sprintf(" Birdies: %d  │  Pars: %d  │  Bogeys: %d  │  Doubles+: %d",
			birdies, pars, bogeys, doubles)) + "\n")
		sb.WriteString(bottom)

		parts = append(parts, sb.String())
	}

	return strings.Join(parts, "\n\n")
}

func (m model) saveScorecard() (tea.Model, tea.Cmd) {
	sc := m.scorecard

	// Build save JSON
	type savePlayer struct {
		Name   string `json:"name"`
		Scores [18]int `json:"scores"`
	}
	type saveCourse struct {
		Name     string  `json:"name"`
		HolePars [18]int `json:"holePars"`
	}
	type savePayload struct {
		Course    saveCourse   `json:"course"`
		Players   []savePlayer `json:"players"`
		ImagePath string       `json:"imagePath"`
	}

	payload := savePayload{
		Course:    saveCourse{Name: sc.CourseName, HolePars: sc.HolePars},
		ImagePath: sc.ImagePath,
	}
	for _, p := range sc.Players {
		payload.Players = append(payload.Players, savePlayer{Name: p.Name, Scores: p.Scores})
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		m.output = errorStyle.Render("Failed to encode scorecard: " + err.Error())
		m.state = stateOutput
		return m, nil
	}

	tmpPath := filepath.Join(os.TempDir(), "scorecard_review.json")
	if err := os.WriteFile(tmpPath, raw, 0644); err != nil {
		m.output = errorStyle.Render("Failed to write temp file: " + err.Error())
		m.state = stateOutput
		return m, nil
	}

	summary := formatRoundSummary(sc)
	m.scorecard = nil
	m.output = summary
	m.state = stateOutput
	return m, func() tea.Msg {
		var errOut bytes.Buffer
		c := exec.Command("python3", "scan.py", "save", tmpPath)
		c.Dir = projectRoot
		c.Stderr = &errOut
		runErr := c.Run()
		if runErr != nil {
			return scorecardSavedMsg{err: fmt.Errorf("%w\n%s", runErr, errOut.String())}
		}
		return scorecardSavedMsg{}
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

const minScorecardWidth = 113

// renderScorecardTable builds the box-drawing scorecard table string.
// Shared by viewScorecard and viewRoundView.
func (m model) renderScorecardTable() string {
	sc := m.scorecard
	var sb strings.Builder

	// ── helpers ──
	isFocused := func(row, col int) bool {
		return !m.editingCell && m.cursor.row == row && m.cursor.col == col
	}
	isEditing := func(row, col int) bool {
		return m.editingCell && m.cursor.row == row && m.cursor.col == col
	}

	renderNum := func(val int, w int, row, col int) string {
		s := fmt.Sprintf("%*d", w, val)
		switch {
		case isEditing(row, col):
			display := m.editBuf
			if display == "" {
				display = s
			} else {
				display = fmt.Sprintf("%*s", w, display)
			}
			return editingCellStyle.Render(display)
		case isFocused(row, col):
			return focusedCellStyle.Render(s)
		default:
			return s
		}
	}

	renderName := func(name string, width int, row int) string {
		s := fmt.Sprintf("%-*s", width, name)
		switch {
		case isEditing(row, -1):
			inp := m.input.View()
			if len(inp) > width {
				inp = inp[:width]
			}
			return editingCellStyle.Render(fmt.Sprintf("%-*s", width, inp))
		case isFocused(row, -1):
			return focusedCellStyle.Render(s)
		default:
			return s
		}
	}

	sum := func(vals [18]int, from, to int) int {
		t := 0
		for i := from; i < to; i++ {
			t += vals[i]
		}
		return t
	}

	parTotal := sum(sc.HolePars, 0, 18)
	parFront := sum(sc.HolePars, 0, 9)
	parBack  := sum(sc.HolePars, 9, 18)

	topBorder    := "┌" + strings.Repeat("─", 98) + "┐"
	header       := fmt.Sprintf("│ COURSE: %-*s PAR: %3d │",
		79, renderCourseName(sc.CourseName, m, isFocused(-1, -1), isEditing(-1, -1)), parTotal)
	colSep       := "├────────┬───┬───┬───┬───┬───┬───┬───┬───┬───╥─────┬───┬───┬───┬───┬───┬───┬───┬───┬───╥─────╥─────┤"
	holeRow      := "│ HOLE   │ 1 │ 2 │ 3 │ 4 │ 5 │ 6 │ 7 │ 8 │ 9 ║ OUT │10 │11 │12 │13 │14 │15 │16 │17 │18 ║ IN  ║ TOT │"
	holeSep      := "├────────┼───┼───┼───┼───┼───┼───┼───┼───┼───╫─────┼───┼───┼───┼───┼───┼───┼───┼───┼───╫─────╫─────┤"
	parPlayerSep := "╞════════╪═══╪═══╪═══╪═══╪═══╪═══╪═══╪═══╪═══╬═════╪═══╪═══╪═══╪═══╪═══╪═══╪═══╪═══╪═══╬═════╬═════╡"
	playerSep    := "├────────┼───┼───┼───┼───┼───┼───┼───┼───┼───╫─────┼───┼───┼───┼───┼───┼───┼───┼───┼───╫─────╫─────┤"
	bottomBorder := "└────────┴───┴───┴───┴───┴───┴───┴───┴───┴───╨─────┴───┴───┴───┴───┴───┴───┴───┴───┴───╨─────╨─────┘"

	parRow := fmt.Sprintf("│ %-6s │%s│%s│%s│%s│%s│%s│%s│%s│%s║%s│%s│%s│%s│%s│%s│%s│%s│%s│%s║%s║%s│",
		"PAR",
		renderNum(sc.HolePars[0], 3, 0, 0), renderNum(sc.HolePars[1], 3, 0, 1),
		renderNum(sc.HolePars[2], 3, 0, 2), renderNum(sc.HolePars[3], 3, 0, 3),
		renderNum(sc.HolePars[4], 3, 0, 4), renderNum(sc.HolePars[5], 3, 0, 5),
		renderNum(sc.HolePars[6], 3, 0, 6), renderNum(sc.HolePars[7], 3, 0, 7),
		renderNum(sc.HolePars[8], 3, 0, 8),
		fmt.Sprintf(" %3d ", parFront),
		renderNum(sc.HolePars[9], 3, 0, 9),   renderNum(sc.HolePars[10], 3, 0, 10),
		renderNum(sc.HolePars[11], 3, 0, 11), renderNum(sc.HolePars[12], 3, 0, 12),
		renderNum(sc.HolePars[13], 3, 0, 13), renderNum(sc.HolePars[14], 3, 0, 14),
		renderNum(sc.HolePars[15], 3, 0, 15), renderNum(sc.HolePars[16], 3, 0, 16),
		renderNum(sc.HolePars[17], 3, 0, 17),
		fmt.Sprintf(" %3d ", parBack),
		fmt.Sprintf(" %3d ", parTotal),
	)

	sb.WriteString(topBorder + "\n")
	sb.WriteString(header + "\n")
	sb.WriteString(colSep + "\n")
	sb.WriteString(holeRow + "\n")
	sb.WriteString(holeSep + "\n")
	sb.WriteString(parRow + "\n")
	sb.WriteString(parPlayerSep + "\n")

	for pi, p := range sc.Players {
		rowIdx := pi + 1
		front  := sum(p.Scores, 0, 9)
		back   := sum(p.Scores, 9, 18)
		total  := front + back

		row := fmt.Sprintf("│ %-6s │%s│%s│%s│%s│%s│%s│%s│%s│%s║%s│%s│%s│%s│%s│%s│%s│%s│%s│%s║%s║%s│",
			renderName(p.Name, 6, rowIdx),
			renderNum(p.Scores[0], 3, rowIdx, 0),   renderNum(p.Scores[1], 3, rowIdx, 1),
			renderNum(p.Scores[2], 3, rowIdx, 2),   renderNum(p.Scores[3], 3, rowIdx, 3),
			renderNum(p.Scores[4], 3, rowIdx, 4),   renderNum(p.Scores[5], 3, rowIdx, 5),
			renderNum(p.Scores[6], 3, rowIdx, 6),   renderNum(p.Scores[7], 3, rowIdx, 7),
			renderNum(p.Scores[8], 3, rowIdx, 8),
			fmt.Sprintf(" %3d ", front),
			renderNum(p.Scores[9], 3, rowIdx, 9),   renderNum(p.Scores[10], 3, rowIdx, 10),
			renderNum(p.Scores[11], 3, rowIdx, 11), renderNum(p.Scores[12], 3, rowIdx, 12),
			renderNum(p.Scores[13], 3, rowIdx, 13), renderNum(p.Scores[14], 3, rowIdx, 14),
			renderNum(p.Scores[15], 3, rowIdx, 15), renderNum(p.Scores[16], 3, rowIdx, 16),
			renderNum(p.Scores[17], 3, rowIdx, 17),
			fmt.Sprintf(" %3d ", back),
			fmt.Sprintf(" %3d ", total),
		)
		sb.WriteString(row + "\n")
		if pi < len(sc.Players)-1 {
			sb.WriteString(playerSep + "\n")
		}
	}
	sb.WriteString(bottomBorder)
	return sb.String()
}

func (m model) viewScorecard() string {
	if m.width < minScorecardWidth {
		msg := fmt.Sprintf("Terminal must be at least %d chars wide (current: %d). Please resize.", minScorecardWidth, m.width)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errorStyle.Render(msg))
	}

	table := m.renderScorecardTable()
	help  := helpStyle.Render("↑/↓/←/→ navigate   enter edit   s save   esc cancel")
	ui    := lipgloss.JoinVertical(lipgloss.Left, table, help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

func renderCourseName(name string, m model, focused, editing bool) string {
	if editing {
		return m.input.View()
	}
	if focused {
		return focusedCellStyle.Render(name)
	}
	return name
}
