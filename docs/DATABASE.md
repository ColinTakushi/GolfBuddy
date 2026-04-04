# Golf Scorecard Tracking System - Database Implementation

A comprehensive user progression tracking system for golf scorecards using SQLite, SQLAlchemy, and FastAPI.

## Overview

The system now includes:
- **SQLite Database** for persistent storage of users, courses, scorecards, and individual hole scores
- **SQLAlchemy ORM** for type-safe database operations
- **FastAPI REST API** for querying user progression, scores, and statistics
- **Data Migration** script to import existing CSV data
- **Enhanced Analytics** with detailed scoring breakdowns and comparisons

## Architecture

### Database Schema

#### Users Table
- **id**: Integer (Primary Key)
- **username**: String (Unique)
- **created_at**: DateTime
- Relationships: One-to-many with Scorecards

#### Courses Table
- **id**: Integer (Primary Key)
- **name**: String (Unique)
- **holes_par**: JSON array of 18 par values
- **created_at**: DateTime
- Relationships: One-to-many with Scorecards

#### Scorecards Table
- **id**: Integer (Primary Key)
- **user_id**: Foreign Key → Users
- **course_id**: Foreign Key → Courses
- **date**: DateTime (when the round was played)
- **image_path**: String (path to scorecard image file)
- **raw_ocr_data**: JSON (raw OCR output for debugging)
- **created_at**: DateTime
- Relationships: One-to-many with Scores

#### Scores Table
- **id**: Integer (Primary Key)
- **scorecard_id**: Foreign Key → Scorecards
- **hole_number**: Integer (1-18)
- **score**: Integer (strokes for that hole)
- **created_at**: DateTime

## Setup & Installation

### 1. Install Dependencies
```bash
source score_card_env/bin/activate
pip install -r requirments.txt
```

Or install specific packages:
```bash
pip install SQLAlchemy==2.0.25 fastapi==0.109.0 uvicorn==0.27.0 pydantic==2.5.3 httpx requests
```

### 2. Initialize Database
The database is automatically created on first import. To manually initialize:
```python
from database import init_db
init_db()
```

### 3. Migrate Existing CSV Data
```bash
python migrate.py
```

This script will:
- Parse `ScoreCard.csv`
- Create a course entry with the Par row
- Create user entries
- Create scorecard entries with individual hole scores
- Populate the SQLite database

## Usage

### Command-Line Analysis (BreakDown.py)

**List all users:**
```bash
python BreakDown.py
```

**Get breakdown for a specific user:**
```bash
python BreakDown.py Colin
```

Output includes:
- Total rounds played
- Average score
- Best/worst scores
- Average differential (vs. par)
- Course breakdown
- Per-round statistics (holes breakdown, front/back 9 scores)

### REST API (api.py)

**Start the API server:**
```bash
python api.py
```

The server runs on `http://localhost:8000` by default.

#### Core Endpoints

**Users Management**
- `GET /` - API info
- `GET /users` - List all users
- `POST /users` - Create new user
  ```bash
  curl -X POST "http://localhost:8000/users?username=NewPlayer"
  ```
- `GET /users/{username}` - Get user stats and progression
  ```bash
  curl "http://localhost:8000/users/Colin"
  ```

**Courses**
- `GET /courses` - List all courses
- `POST /courses` - Create new course
  ```bash
  curl -X POST "http://localhost:8000/courses?name=Pebble Beach&holes_par=[4,4,4,3,4,5,3,4,4,4,4,4,3,4,4,3,4,5]"
  ```
- `GET /courses/{course_id}` - Get course details and stats

**Scorecards**
- `GET /scorecards/{username}` - Get all scorecards for a user
  ```bash
  curl "http://localhost:8000/scorecards/Colin"
  ```
- `POST /scorecards` - Add a new scorecard
  ```bash
  curl -X POST "http://localhost:8000/scorecards?username=Colin&course_id=1&scores=[4,5,6,4,7,5,5,7,6,5,6,6,5,4,5,5,5,4]&image_path=/path/to/image.jpg"
  ```
- `GET /scorecards/{username}/{scorecard_id}` - Get detailed scorecard with all holes

**Statistics**
- `GET /stats/{username}` - Get aggregated user statistics
  ```bash
  curl "http://localhost:8000/stats/Colin"
  ```
  Returns: total rounds, average score, best/worst, trend over time, course breakdown

- `GET /stats/{username}?days=30` - Get stats for last N days
  ```bash
  curl "http://localhost:8000/stats/Colin?days=30"
  ```

- `GET /stats/{username}/course/{course_id}` - Get course-specific stats
  ```bash
  curl "http://localhost:8000/stats/Colin/course/1"
  ```

#### Response Examples

**User Stats:**
```json
{
  "id": 3,
  "username": "Colin",
  "created_at": "2026-04-04T16:29:38.123050",
  "total_rounds": 1,
  "average_score": 98.0,
  "best_score": 98,
  "worst_score": 98,
  "courses_played": 1
}
```

**User Scorecard Summary:**
```json
[
  {
    "id": 3,
    "course": "Championship Course",
    "date": "2024-01-03T00:00:00",
    "total_score": 98,
    "total_par": 70,
    "score_differential": 28,
    "front_9": 52,
    "back_9": 46,
    "hole_breakdown": {
      "birdies": 0,
      "pars": 5,
      "bogeys": 5,
      "doubles_plus": 8
    },
    "image_path": null
  }
]
```

## Integration with OCR System

The `ocr_utils.py` module provides functions to save OCR results to the database:

```python
from database import SessionLocal
from ocr_utils import save_scorecard_to_db, store_ocr_result

db = SessionLocal()

# Store OCR data from EasyOCR output
ocr_result = model.readtext(image_path, detail=1)
parsed_data = store_ocr_result(ocr_result)

# Save scorecard to database with image
scorecard = save_scorecard_to_db(
    db=db,
    username="Colin",
    course_name="Championship Course",
    scores=[4, 5, 6, 4, 7, 5, 5, 7, 6, 5, 6, 6, 5, 4, 5, 5, 5, 4],
    course_pars=[4, 4, 4, 3, 4, 5, 3, 4, 4, 4, 4, 4, 3, 4, 4, 3, 4, 5],
    image_path="scorecard_image.jpg",
    raw_ocr_data=parsed_data
)

db.close()
```

Images are stored in: `images/users/{username}/scorecards/{timestamp}.jpg`

## Key Features

### Per-User Tracking
- Track multiple rounds per user
- Multiple courses with different par values
- Detailed scoring breakdown (birdies, pars, bogeys, doubles)
- Front 9 vs Back 9 analysis
- Progression over time

### Course Management
- Store custom courses with specific hole par values
- Track average scores per course
- View all players' statistics at each course

### Advanced Statistics
- Average score calculation
- Best/worst round tracking
- Course-specific averages
- Time-based filtering (optional)
- Hole breakdown analysis

### Image Storage
- Efficient file-based storage (not BLOBs)
- Organized by user and date
- Optional raw OCR data for debugging

## File Structure

```
score-card-reader/
├── database.py           # SQLAlchemy setup and session management
├── models.py             # ORM model definitions (User, Course, Scorecard, Score)
├── migrate.py            # Script to import CSV data to database
├── api.py                # FastAPI REST API server
├── ocr_utils.py          # Utilities for saving OCR results
├── BreakDown.py          # Command-line analysis tool (updated for database)
├── ocr_test.py           # OCR processing (can integrate with database)
├── scorecard.db          # SQLite database file (auto-created)
├── images/               # Image storage directory
│   └── users/
│       └── {username}/
│           └── scorecards/
├── ScoreCard.csv         # Original scorecard data (for migration)
└── requirments.txt       # Python dependencies
```

## Backend Technology Recommendations

### SQLite (Chosen)
✅ **Pros:**
- No server setup required
- File-based (easy to backup/transfer)
- Perfect for single-user or small team scenarios
- Easy to query with Python
- Excellent for prototyping

❌ **Cons:**
- Limited concurrent writes
- Not ideal for high-traffic scenarios

### PostgreSQL (Alternative)
✅ **Pros:**
- Better concurrent access
- More robust for production
- Advanced query capabilities
- Good for multi-user scenarios

❌ **Cons:**
- Requires server setup
- More complex deployment

### MongoDB (Alternative)
✅ **Pros:**
- Flexible schema (good for varying scorecard formats)
- Good for NoSQL scenarios

❌ **Cons:**
- Larger data files
- Less suitable for relational queries
- Overkill for this use case

### Recommendation
**SQLite** is the best choice for this application because:
1. Simple, file-based storage
2. No external server needed
3. Good performance for your use case
4. Easy integration with Python
5. Can migrate to PostgreSQL later if needed

## Future Enhancements

1. **Web UI** - Add a Flask/Django frontend for visual data exploration
2. **Authentication** - User login system for multi-user support
3. **Handicap Calculation** - Automated handicap tracking
4. **Leaderboards** - Per-course and historical leaderboards
5. **Export** - CSV/PDF report generation
6. **Cloud Sync** - Backup to cloud storage
7. **Mobile App** - Direct scorecard entry from mobile devices
8. **Advanced Analytics** - Performance trends, weaknesses by hole type
9. **Course Database** - Integration with real golf course databases
10. **Alerts** - Notifications for personal bests or milestones

## Troubleshooting

### Database Already Locked
If you see "database is locked" errors:
- Ensure only one process is writing to the database
- SQLite has limited concurrent write support
- Consider PostgreSQL for higher concurrency

### Migration Fails
- Ensure `ScoreCard.csv` exists in the project root
- Check CSV format matches expected structure
- Delete `scorecard.db` and re-run `migrate.py`

### API Port Already in Use
```bash
# Kill process on port 8000
lsof -ti:8000 | xargs kill -9
```

## License

This project is part of the golf scorecard OCR reader system.
