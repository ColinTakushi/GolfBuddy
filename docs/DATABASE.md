# Golf Scorecard Tracking System - Database

## Overview

- **SQLite** for persistent storage via SQLAlchemy ORM
- **FastAPI** REST API for querying user progression and statistics
- **Gemini 2.5 Flash** for OCR image processing

## Database Schema

### Users
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary Key |
| username | String | Unique |
| created_at | DateTime | |

### Courses
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary Key |
| name | String | Unique |
| holes_par | JSON | Array of 18 par values |
| created_at | DateTime | |

### Scorecard Images
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary Key |
| path | String | Path to stored image file |
| created_at | DateTime | |

One image record is shared across all scorecards created from the same scan.

### Scorecards
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary Key |
| user_id | Integer | FK → Users |
| course_id | Integer | FK → Courses |
| image_id | Integer | FK → Scorecard Images (nullable) |
| date | DateTime | When the round was played |
| raw_ocr_data | JSON | Raw Gemini output |
| created_at | DateTime | |

### Scores
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary Key |
| scorecard_id | Integer | FK → Scorecards |
| hole_number | Integer | 1–18 |
| score | Integer | Strokes for that hole |
| created_at | DateTime | |

## Setup

```bash
python3 -m venv score_card_env
source score_card_env/bin/activate
pip install -r requirements.txt
```

The database is created automatically at `data/db/scorecard.db` on first run.

## CLI Usage

```bash
python main.py api               # Start REST API server
python main.py stats <username>  # Show stats for a user
python main.py stats             # List all users
python main.py migrate           # Import CSV data
python main.py nuke              # Clear all data (testing)
```

## API Endpoints

**Users**
- `GET /users` — List all users
- `POST /users?username=Colin` — Create a user
- `GET /users/{username}` — User detail and stats

**Courses**
- `GET /courses` — List all courses
- `POST /courses?name=...&holes_par=[...]` — Create a course
- `GET /courses/{course_id}` — Course detail and stats

**Scorecards**
- `GET /scorecards/{username}` — All scorecards for a user
- `GET /scorecards/{username}/{scorecard_id}` — Detailed scorecard with hole breakdown
- `POST /scorecards` — Create a scorecard manually

**Statistics**
- `GET /stats/{username}` — Aggregated stats (total rounds, average, best/worst, trend)
- `GET /stats/{username}?days=30` — Stats filtered to last N days
- `GET /stats/{username}/course/{course_id}` — Course-specific stats

Full interactive docs available at `http://localhost:8000/docs`.

## Image Storage

Scorecard images are stored once per scan at:
```
images/scorecards/scorecard_{timestamp}.jpg
```

All scorecards (one per player) from the same scan share the same `image_id`.

## Troubleshooting

### API port already in use
```bash
lsof -ti:8000 | xargs kill -9
```

### Reset the database
```bash
python main.py nuke
```
