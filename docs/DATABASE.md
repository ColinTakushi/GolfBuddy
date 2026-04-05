# Database

## Overview

- **SQLite** at `data/db/scorecard.db`
- **SQLAlchemy ORM** — models in `src/core/models.py`
- Database is created automatically on first run

## Schema

### Users
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| username | String | Unique |
| created_at | DateTime | |

### Courses
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| name | String | Unique |
| holes_par | JSON | Array of 18 par values |
| created_at | DateTime | |

### Scorecard Images
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| path | String | Path under `images/scorecards/` |
| created_at | DateTime | |

One image record is shared across all scorecards created from the same scan.

### Scorecards
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| user_id | Integer | FK → Users |
| course_id | Integer | FK → Courses |
| image_id | Integer | FK → Scorecard Images (nullable) |
| date | DateTime | When the round was played |
| raw_ocr_data | JSON | Raw Gemini output (nullable) |
| created_at | DateTime | |

### Scores
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| scorecard_id | Integer | FK → Scorecards |
| hole_number | Integer | 1–18 |
| score | Integer | Strokes for that hole |
| created_at | DateTime | |

## Scorecard Methods

`Scorecard` ORM objects expose these helper methods:

| Method | Returns |
|--------|---------|
| `get_total_score()` | Sum of all 18 hole scores |
| `get_total_par()` | Total par from the course |
| `get_score_differential()` | `total_score - total_par` |
| `get_front_9_score()` | Sum of holes 1–9 |
| `get_back_9_score()` | Sum of holes 10–18 |
| `get_hole_breakdown()` | Dict with counts of birdies, pars, bogeys, doubles+ |

## API Endpoints

**Users**
- `GET /users` — list all users with round counts
- `POST /users?username=Colin` — create a user
- `GET /users/{username}` — user detail and aggregate stats

**Courses**
- `GET /courses` — list all courses
- `POST /courses?name=...&holes_par=[...]` — create a course
- `GET /courses/{course_id}` — course detail

**Scorecards**
- `GET /scorecards/{username}` — all rounds for a user (newest first)
- `GET /scorecards/{username}/{scorecard_id}` — full scorecard with per-hole breakdown
- `POST /scorecards` — create a scorecard manually
- `PUT /scorecards/{scorecard_id}` — replace all 18 hole scores for an existing round

**Statistics**
- `GET /stats/{username}` — aggregated stats (total rounds, average, best/worst, trend, handicap estimate)
- `GET /stats/{username}?days=30` — stats filtered to last N days
- `GET /stats/{username}/course/{course_id}` — stats at a specific course

Full interactive docs: `http://localhost:8000/docs`

## Image Storage

Scorecard images are stored once per scan at:
```
images/scorecards/scorecard_{timestamp}.jpg
```

All player scorecards from the same scan share a single `image_id`.

## Reset the Database

```bash
python main.py nuke
```

Deletes `data/db/scorecard.db` and recreates the schema from scratch.
