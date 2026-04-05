# Database

## Overview

- **SQLite** at `data/db/scorecard.db`
- **SQLAlchemy ORM** — models in `src/core/models.py` and `src/database/models.py` (kept in sync)
- Database is created automatically on first run via `init_db()`

---

## Entity Relationship Diagram

```
┌─────────────────────┐
│        users        │
├─────────────────────┤
│ id (PK)             │
│ username (unique)   │
│ created_at          │
└────────┬────────────┘
         │ M
         │ participates in
         │ M
┌────────▼────────────┐    ┌──────────────────────┐
│ scorecard_players   │    │       courses        │
│   (join table)      │    ├──────────────────────┤
├─────────────────────┤    │ id (PK)              │
│ scorecard_id (FK)   │    │ name (unique)        │
│ user_id (FK)        │    │ hole_pars (JSON)     │
└────────┬────────────┘    │ total_par            │
         │                 │ created_at           │
         │ M               └──────────┬───────────┘
         │ one per round              │ 1
┌────────▼────────────┐               │
│      scorecards     ├───────────────┘
├─────────────────────┤  belongs to 1 course
│ id (PK)             │
│ course_id (FK)      │
│ image_path (nullable)│
│ raw_ocr_data (JSON) │
│ date_played         │
│ created_at          │
│ scores (JSON)       │ ← {"user_id": [int×18], ...}
│ hole_layout (JSON)  │ ← [int×18] pars at time of play
│ total_par           │
└─────────────────────┘
```

### One Round = One Scorecard

A single `Scorecard` represents one round of golf. All players who participated are linked via the `scorecard_players` join table, and each player's 18 hole scores are stored in the `scores` JSON column keyed by their user ID.

```
 Scan one scorecard image
         │
         ▼
  scorecards (1 row)
    scores: {
      "1": [4,5,3,...],   ← Player A (user id=1)
      "2": [5,4,4,...],   ← Player B (user id=2)
    }
    hole_layout: [4,4,3,...]
    total_par: 72
         │
         ▼
  scorecard_players
    (scorecard_id=7, user_id=1)
    (scorecard_id=7, user_id=2)
```

---

## Schema

### users
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| username | String | Unique, indexed |
| created_at | DateTime | |

### courses
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| name | String | Unique, indexed |
| hole_pars | JSON | Array of 18 par values e.g. `[4,4,3,5,...]` |
| total_par | Integer | Stored sum of hole_pars |
| created_at | DateTime | |

### scorecard_players *(join table)*
| Column | Type | Notes |
|--------|------|-------|
| scorecard_id | Integer | FK → scorecards, composite PK |
| user_id | Integer | FK → users, composite PK |

### scorecards
| Column | Type | Notes |
|--------|------|-------|
| id | Integer | Primary key |
| course_id | Integer | FK → courses |
| image_path | String | Path to scorecard image (nullable) |
| raw_ocr_data | JSON | Raw Gemini OCR output (nullable, for debugging) |
| date_played | DateTime | When the round was played |
| created_at | DateTime | When the record was created |
| scores | JSON | `{"user_id": [int×18], ...}` — all players' scores |
| hole_layout | JSON | `[int×18]` — par values copied from course at scan time |
| total_par | Integer | Sum of hole_layout |

---

## Scorecard Helper Methods

All score methods require a `user_id` since scores are stored per-player:

| Method | Returns |
|--------|---------|
| `get_total_score(user_id)` | Sum of the player's 18 hole scores |
| `get_front_9_score(user_id)` | Sum of holes 1–9 |
| `get_back_9_score(user_id)` | Sum of holes 10–18 |
| `get_score_differential(user_id)` | `total_score - total_par` |
| `get_hole_breakdown(user_id)` | Dict: `{birdies, pars, bogeys, doubles_plus}` counts |

`Course` helper methods (no user_id needed): `get_front_9_par()`, `get_back_9_par()`

---

## API Endpoints

**Users**
- `GET /users` — list all users with round counts
- `POST /users?username=Colin` — create a user
- `GET /users/{username}` — user detail and aggregate stats

**Courses**
- `GET /courses` — list all courses with round counts
- `POST /courses?name=...&holes_par=[...]` — create a course (exactly 18 par values)
- `GET /courses/{course_id}` — course detail, average score across all players

**Scorecards**
- `GET /scorecards/{username}` — all rounds a user participated in, newest first
- `GET /scorecards/{username}/{scorecard_id}` — full per-hole breakdown for that user
- `POST /scorecards` — create a scorecard for a full round (body below)
- `PUT /scorecards/{scorecard_id}?username=Colin` — replace a player's 18 scores

`POST /scorecards` request body:
```json
{
  "course": { "name": "Pebble Beach", "holePars": [4,4,3,5,3,5,3,4,4,4,4,3,4,4,5,4,3,5] },
  "players": [
    { "name": "Colin", "scores": [4,5,3,6,3,5,3,4,5,4,4,3,4,4,5,4,3,5] },
    { "name": "Alice", "scores": [5,4,4,5,3,5,4,4,4,4,5,3,5,4,5,4,4,5] }
  ],
  "imagePath": "data/images/scorecard_20260405.jpg"
}
```

**Statistics**
- `GET /stats/{username}` — rounds, avg, best/worst, trend, per-course breakdown, handicap estimate
- `GET /stats/{username}?days=30` — same, filtered to last N days
- `GET /stats/{username}/course/{course_id}` — stats at a specific course

**Admin**
- `DELETE /nuke` — delete all data (scorecard_players → scorecards → users → courses)

Full interactive docs: `http://localhost:8000/docs`

---

## Delete Behavior

When a user is deleted:
- Their row is removed from `scorecard_players`
- Their key is removed from `scorecard.scores` for affected rounds
- The Scorecard itself stays (other players' data is preserved)

Only `DELETE /nuke` removes everything.

---

## Image Storage

Scorecard images are stored under:
```
data/images/
```

The path is stored directly as `scorecards.image_path` (nullable string). One image per scan is shared across all players in that round via the single Scorecard row.

---

## Reset the Database

```bash
python main.py nuke
```

Drops all rows in dependency order and recreates the schema. Equivalent to `DELETE /nuke` via the API.
