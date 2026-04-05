# Quick Start

## Prerequisites

- Python 3.12+
- Go 1.21+
- `GEMINI_API_KEY` environment variable

## Setup

```bash
python3 -m venv score_card_env
source score_card_env/bin/activate
pip install -r requirements.txt
export GEMINI_API_KEY=<your_key>
```

## Run the TUI

```bash
cd tui
go mod tidy   # first time only
go run .
```

The REST API starts automatically in the background. Navigate with `↑/↓`, select with `Enter`, go back with `Esc`.

---

## TUI Flows

### Scan → Review → Save

1. Select **scan → image**, enter the image path
2. Gemini extracts the scorecard; the table editor opens
3. Navigate cells with arrow keys, press `e` or `Enter` to edit a value, digits to type, `Enter`/arrow to confirm
4. Press `s` to save — one scorecard per player is written to the database

To skip Gemini (testing): **scan → json**, enter a path to a pre-parsed JSON file.

### Stats Browser

1. Select **stats** — a list of all players loads
2. Select a player to see their aggregate stats and round history
3. Select a round to view its full scorecard
4. Press `e` to edit scores, `s` to save changes back to the database
5. `Esc` returns to the previous screen

---

## REST API (manual)

```bash
python main.py api   # starts on http://localhost:8000
```

```bash
# Users
curl http://localhost:8000/users
curl http://localhost:8000/users/Colin

# Scorecards
curl http://localhost:8000/scorecards/Colin
curl http://localhost:8000/scorecards/Colin/3

# Stats
curl http://localhost:8000/stats/Colin
curl "http://localhost:8000/stats/Colin?days=30"

# Update scores for a round
curl -X PUT http://localhost:8000/scorecards/3 \
  -H "Content-Type: application/json" \
  -d '[4,3,5,4,4,4,4,3,5,4,4,4,3,4,5,4,4,4]'
```

Interactive docs: `http://localhost:8000/docs`

---

## CLI (direct)

```bash
python main.py stats             # list all users
python main.py stats Colin       # stats for a specific user
python main.py nuke              # delete all data and recreate schema
```

---

## File Locations

| Item | Location |
|------|----------|
| Database | `data/db/scorecard.db` |
| Scorecard images | `images/scorecards/` |
| ORM models | `src/core/models.py` |
| API server | `src/api/server.py` |

## Troubleshooting

**API port 8000 already in use**
```bash
lsof -ti:8000 | xargs kill -9
```

**Reset the database**
```bash
python main.py nuke
```
