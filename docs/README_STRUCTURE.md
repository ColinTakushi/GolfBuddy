# Project Structure

```
GolfBuddy/
├── src/
│   ├── core/               # SQLAlchemy engine and ORM models (used by API)
│   │   ├── db.py           # Engine, SessionLocal, get_db, init_db
│   │   └── models.py       # User, Course, ScorecardImage, Scorecard, Score
│   ├── database/           # CLI-facing mirrors of src/core/ (same DB, same schema)
│   │   ├── db.py
│   │   └── models.py
│   ├── api/
│   │   ├── server.py       # FastAPI app — used at runtime (imports src.core)
│   │   └── main.py         # FastAPI app — used by CLI (imports src.database)
│   └── ocr/
│       ├── pipeline.py     # Gemini image → JSON; save_from_data (no prompts)
│       └── utils.py        # save_image_to_db, save_scorecard_to_db
├── tools/
│   └── analytics.py        # print_round_summary, print_user_breakdown
├── tui/
│   ├── go.mod              # Go module (bubbletea, bubbles, lipgloss)
│   ├── main.go             # Entry point — starts API server in background, runs TUI
│   ├── model.go            # Bubbletea model: state machine, menus, Update, View
│   ├── scorecard.go        # Scorecard table editor (scan flow + round view rendering)
│   ├── stats.go            # Stats browser: player list, round list, round view
│   └── styles.go           # Lipgloss styles
├── data/
│   └── db/
│       └── scorecard.db    # SQLite database
├── images/
│   └── scorecards/         # Scorecard images (one file per scan, shared across players)
├── docs/
│   ├── DATABASE.md         # Schema, API endpoints, image storage
│   └── QUICKSTART.md       # Setup and usage guide
├── scan.py                 # Python CLI: scan image / load JSON / save reviewed data
├── main.py                 # Python CLI: api / stats / nuke
└── requirements.txt
```

## TUI State Machine

| State | Description |
|-------|-------------|
| `stateMainMenu` | Top-level menu (scan, stats, nuke, close) |
| `stateSubMenu` | Submenu for the selected item |
| `stateInput` | Text input with optional file-path autocomplete |
| `stateOutput` | Displays command output or round summary |
| `stateScorecard` | Scorecard table editor (after scanning) |
| `statePlayerList` | Browsable list of all players |
| `statePlayerDetail` | Aggregate stats + round list for a player |
| `stateRoundView` | Full scorecard for a historical round (view/edit) |

## Module Summary

### `src/core/` and `src/database/`
Mirror modules — both use the same `DATABASE_URL` from `src/config.py` and the same schema. `src/core/` is imported by the API server; `src/database/` by CLI entry points.

### `src/ocr/pipeline.py`
- `parse_image(path)` — calls Gemini, returns JSON dict (no prompts, no save)
- `save_from_data(data)` — saves course + players to DB from a dict (no prompts)

### `src/api/server.py`
FastAPI app with endpoints for users, courses, scorecards, and statistics. Includes `PUT /scorecards/{id}` for updating hole scores.

### `tools/analytics.py`
- `print_round_summary(scorecard)` — formatted single-round stats box
- `print_user_breakdown(username)` — full history summary for a user

### `tui/scorecard.go`
- `scorecardData` / `playerData` — in-memory scorecard representation
- `renderScorecardTable()` — shared table renderer (used by both scan and round view)
- `saveScorecard()` — writes JSON to `/tmp/`, calls `scan.py save`
- `formatRoundSummary(sc)` — builds the post-save stats box

### `tui/stats.go`
- HTTP fetch commands: `cmdFetchPlayers`, `cmdFetchRounds`, `cmdFetchRoundDetail`, `cmdUpdateRound`
- Update handlers: `updatePlayerList`, `updatePlayerDetail`, `updateRoundView`
- Views: `viewPlayerList`, `viewPlayerDetail`, `viewRoundView`

## Key Imports

```python
from src.core.db import SessionLocal, get_db, init_db
from src.core.models import User, Course, ScorecardImage, Scorecard, Score
from src.ocr.pipeline import parse_image, save_from_data
from src.ocr.utils import save_image_to_db, save_scorecard_to_db
from tools.analytics import print_round_summary, print_user_breakdown
```
