# Quick Start Guide

## Setup

```bash
python3 -m venv score_card_env
source score_card_env/bin/activate
pip install -r requirements.txt
export GEMINI_API_KEY=<your_key>
```

## Scan a Scorecard Image

```bash
python main.py scan golf.jpg
```

Sends the image to Gemini 2.5 Flash, displays extracted course and player data, walks you through confirming or correcting each section, then saves one scorecard per player to the database.

## Stats (Command Line)

```bash
python main.py stats            # list all users
python main.py stats Colin      # stats for a specific user
```

## REST API

```bash
python main.py api              # starts on http://localhost:8000
```

```bash
# Users
curl http://localhost:8000/users
curl http://localhost:8000/users/Colin
curl -X POST "http://localhost:8000/users?username=JohnDoe"

# Scorecards
curl http://localhost:8000/scorecards/Colin
curl http://localhost:8000/scorecards/Colin/3

# Stats
curl http://localhost:8000/stats/Colin
curl "http://localhost:8000/stats/Colin?days=30"
curl http://localhost:8000/stats/Colin/course/1

# Courses
curl http://localhost:8000/courses
curl -X POST "http://localhost:8000/courses?name=MyGolfCourse&holes_par=[4,4,4,3,4,5,3,4,4,4,4,4,3,4,4,3,4,5]"
```

Interactive API docs: `http://localhost:8000/docs`

## Reset Database (Testing)

```bash
python main.py nuke
```

## File Locations

| Item | Location |
|------|----------|
| Database | `data/db/scorecard.db` |
| Scorecard Images | `images/scorecards/` |
| Models | `src/core/models.py` |
| API Server | `src/api/server.py` |

## Troubleshooting

### "User not found"
Check spelling — query `/users` to see available users.

### API port 8000 already in use
```bash
lsof -ti:8000 | xargs kill -9
```
