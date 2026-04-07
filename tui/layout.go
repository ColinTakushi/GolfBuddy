package main

// All visual dimensions live here. Change a constant and the whole UI adapts.

const (
	// ── Main menu / input panels ──────────────────────────────────────────────
	menuLeftWidth  = 28
	menuRightWidth = 99

	menuTitleWidth = menuLeftWidth + menuRightWidth + 9 // renders to 140
	menuBodyWidth  = menuLeftWidth + menuRightWidth + 7 // renders to 140

	// ── Stats browser ─────────────────────────────────────────────────────────
	statsPanelWidth  = 70
	statsInner       = statsPanelWidth - 2 // inner content width (between borders)
	statsDateCol     = 12                  // width of the date column
	statsCourseCol   = 28                  // width of the course name column

	// ── Scorecard table ───────────────────────────────────────────────────────
	// scNameColWidth is the only value you need to change to resize the name
	// column; scTableInner and scCourseNameWidth are derived automatically.
	scNameColWidth    = 8
	scTableInner      = scNameColWidth + 92 // inner dashes of the top/bottom border
	scCourseNameWidth = scTableInner - 19   // chars available for the course name

	minScorecardWidth = 115 // minimum terminal width to render the scorecard
)
