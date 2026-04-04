# Golf Scorecard Tracker - Project Structure

## ✨ New Organized Structure

The project has been reorganized into a clean, modular architecture with clear separation of concerns.

```
score-card-reader/
├── src/                          # Main source code
│   ├── __init__.py
│   ├── core/                     # Core database and models
│   │   ├── __init__.py
│   │   ├── db.py                 # Database setup & session management
│   │   └── models.py             # SQLAlchemy ORM models
│   ├── database/                 # Database migration and utilities
│   │   ├── __init__.py
│   │   ├── migration.py          # CSV data migration
│   │   └── models.py             # (legacy - use src/core/models.py)
│   ├── api/                      # REST API server
│   │   ├── __init__.py
│   │   ├── server.py             # FastAPI application
│   │   └── main.py               # (legacy - use server.py)
│   └── ocr/                      # OCR processing
│       ├── __init__.py
│       ├── processor.py          # Image processing & OCR
│       └── utils.py              # OCR utilities & storage
├── tools/                        # Command-line tools
│   ├── __init__.py
│   └── analytics.py              # Scorecard analysis tool
├── images/                       # User scorecard images
│   └── users/{username}/scorecards/
├── tmp/                          # Temporary OCR outputs
├── score_card_env/              # Python virtual environment
├── scorecard.db                 # SQLite database
│
├── QUICKSTART.md                # Quick reference guide (OLD)
├── DATABASE.md                  # API documentation (OLD)
│
├── analyze.py                   # CLI analysis wrapper
├── server.py                    # API server wrapper
├── run_migration.py             # Migration wrapper
│
├── ScoreCard.csv                # Original scorecard data
├── requirments.txt              # Python dependencies
├── main.py                      # (legacy - use analyze.py or server.py)
└── README_STRUCTURE.md          # This file
```

## 🎯 Module Organization

### `src/core/` - Core Database Layer
- **db.py**: Database engine, session factory, and initialization
  - `SessionLocal`: Create database sessions
  - `init_db()`: Initialize database tables
  - `get_db()`: Dependency injection for FastAPI

- **models.py**: SQLAlchemy ORM models
  - `User`: Golfer profiles
  - `Course`: Golf courses with par data
  - `Scorecard`: Individual rounds
  - `Score`: Hole-by-hole scores

### `src/database/` - Data Management
- **migration.py**: Migrate from CSV to SQLite
  - `migrate_from_csv()`: Import `ScoreCard.csv` data
  - Handles users, courses, and scorecards

### `src/api/` - REST API
- **server.py**: FastAPI application
  - `/users` - User management endpoints
  - `/courses` - Course management
  - `/scorecards` - Scorecard CRUD operations
  - `/stats` - Advanced statistics

### `src/ocr/` - OCR Processing
- **processor.py**: Image preprocessing and OCR
  - `process_scorecard_image()`: Prepare image
  - `perform_ocr()`: EasyOCR processing
  - `extract_scorecard_data()`: Parse results

- **utils.py**: Database storage utilities
  - `save_scorecard_to_db()`: Store OCR results
  - `create_image_storage_dir()`: Organize images

### `tools/` - Command-Line Tools
- **analytics.py**: Scorecard statistics
  - `get_user_breakdown()`: Statistics dictionary
  - `print_user_breakdown()`: Formatted output
  - Can be run as: `python -m tools.analytics <username>`

## 📋 Quick Usage Guide

### Run Analysis (NEW)
```bash
python analyze.py <username>
python analyze.py Colin
```

### Start API Server (NEW)
```bash
python server.py              # Port 8000 (default)
python server.py 8080         # Custom port
```

### Run Database Migration (NEW)
```bash
python run_migration.py
```

### Use Python Imports (NEW)
```python
from src.core.db import SessionLocal
from src.core.models import User, Scorecard
from src.api.server import app
from src.ocr.processor import perform_ocr
from tools.analytics import get_user_breakdown
```

## 🔄 Update Guide for Old Code

If you have existing code using the old imports, update them as follows:

### Old → New
```python
# Old
from database import SessionLocal
from models import User

# New
from src.core.db import SessionLocal
from src.core.models import User
```

```python
# Old
from api import app
from api import create_user

# New
from src.api.server import app
```

```python
# Old
from migrate import migrate_from_csv

# New
from src.database.migration import migrate_from_csv
```

```python
# Old
from ocr_utils import save_scorecard_to_db

# New
from src.ocr.utils import save_scorecard_to_db
```

```python
# Old
from BreakDown import get_user_breakdown

# New
from tools.analytics import get_user_breakdown
```

## 🏗️ Architecture Benefits

1. **Separation of Concerns**: Each module has a single responsibility
2. **Scalability**: Easy to add new features without affecting existing code
3. **Testability**: Isolated modules are easier to unit test
4. **Maintainability**: Clear structure makes code easier to navigate
5. **Reusability**: Modules can be imported and used in different contexts

## 📦 Dependency Graph

```
src/core/       (Foundation - no dependencies on other src modules)
  ↓
src/database/   (Imports from src/core)
  ↓
src/api/        (Imports from src/core)
src/ocr/        (Imports from src/core)
  ↓
tools/          (Imports from src/core, src/api, src/ocr)
  ↓
Root wrappers   (analyze.py, server.py, run_migration.py)
```

## 🚀 Future Enhancements

As the project grows, you can:
1. Add `src/auth/` for user authentication
2. Add `src/analysis/` for advanced statistics
3. Add `src/export/` for report generation
4. Add `tests/` for unit and integration tests
5. Add `src/config/` for configuration management

## ❓ Questions?

Refer to:
- `DATABASE.md` - Full API documentation
- `QUICKSTART.md` - Quick reference
- Docstrings in each module for function details
