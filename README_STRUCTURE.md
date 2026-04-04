# Project Structure

```
GolfBuddy/
├── src/
│   ├── core/               # Database engine and ORM models
│   │   ├── db.py           # Engine, SessionLocal, get_db, init_db
│   │   └── models.py       # User, Course, ScorecardImage, Scorecard, Score
│   ├── database/           # Migration utilities
│   │   ├── db.py           # Mirrors src/core/db.py (used by CLI)
│   │   ├── models.py       # Mirrors src/core/models.py (used by CLI)
│   │   └── migrate.py      # Import CSV data into the database
│   ├── api/
│   │   ├── server.py       # FastAPI app (used by API server)
│   │   └── main.py         # FastAPI app (used by CLI)
│   ├── ocr/
│   │   ├── pipeline.py     # Gemini 2.5 Flash image → JSON → DB
│   │   └── utils.py        # save_image_to_db, save_scorecard_to_db
│   └── analysis/
│       └── stats.py        # print_user_breakdown, print_all_users
├── tools/
│   └── analytics.py        # CLI analytics (print_user_breakdown)
├── data/
│   └── db/
│       └── scorecard.db    # SQLite database
├── images/
│   └── scorecards/         # Stored scorecard images (one per scan)
├── docs/
│   ├── DATABASE.md
│   └── QUICKSTART.md
├── main.py                 # CLI entry point (api / stats / migrate / nuke)
├── analyze.py              # Wrapper: python analyze.py <username>
├── server.py               # Wrapper: python server.py [port]
└── requirements.txt
```

## Module Summary

### `src/core/`
- `db.py` — database engine, `SessionLocal`, `get_db`, `init_db`
- `models.py` — ORM models: `User`, `Course`, `ScorecardImage`, `Scorecard`, `Score`

### `src/ocr/`
- `pipeline.py` — full scan pipeline: calls Gemini, prompts user to confirm, saves to DB
- `utils.py` — `save_image_to_db()`, `save_scorecard_to_db()`

### `src/api/`
- FastAPI endpoints for users, courses, scorecards, and statistics

### `tools/analytics.py`
- `print_user_breakdown(username)` — formatted stats output

## CLI Commands

```bash
python main.py api               # Start REST API server
python main.py stats [username]  # List users or show stats
python main.py migrate           # Import CSV data
python main.py nuke              # Clear all data (testing)
```

## Key Imports

```python
from src.core.db import SessionLocal, get_db, init_db
from src.core.models import User, Course, ScorecardImage, Scorecard, Score
from src.ocr.pipeline import scan_and_store
from src.ocr.utils import save_image_to_db, save_scorecard_to_db
from tools.analytics import print_user_breakdown
```
