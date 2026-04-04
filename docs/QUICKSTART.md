# Quick Start Guide - Golf Scorecard Database System

## Activate Virtual Environment
```bash
source score_card_env/bin/activate
```

## Import Existing Data
```bash
# One-time setup: Import ScoreCard.csv into database
python migrate.py
```

## Analyze Scores (Command Line)

### View all users
```bash
python BreakDown.py
```

### View user stats
```bash
python BreakDown.py <username>
```

Examples:
```bash
python BreakDown.py Colin
python BreakDown.py Matty
python BreakDown.py TestPlayer
```

## Use the REST API

### Start the server
```bash
python api.py
```

### Query users (in another terminal)
```bash
# List all users
curl http://localhost:8000/users

# Get specific user stats
curl http://localhost:8000/users/Colin

# Get user progression over time
curl http://localhost:8000/stats/Colin

# Get stats for specific course
curl http://localhost:8000/stats/Colin/course/1

# Filter stats by days
curl "http://localhost:8000/stats/Colin?days=30"
```

### Manage scorecards
```bash
# View all user scorecards
curl http://localhost:8000/scorecards/Colin

# View detailed single scorecard
curl http://localhost:8000/scorecards/Colin/3

# Add new scorecard
curl -X POST "http://localhost:8000/scorecards?username=Colin&course_id=1&scores=[4,5,6,4,7,5,5,7,6,5,6,6,5,4,5,5,5,4]"
```

### Manage courses
```bash
# List courses
curl http://localhost:8000/courses

# Create new course
curl -X POST "http://localhost:8000/courses?name=MyGolfCourse&holes_par=[4,4,4,3,4,5,3,4,4,4,4,4,3,4,4,3,4,5]"

# Get course details
curl http://localhost:8000/courses/1
```

### Manage users
```bash
# Create new user
curl -X POST "http://localhost:8000/users?username=JohnDoe"
```

## Add Scores from OCR

```python
from database import SessionLocal
from ocr_utils import save_scorecard_to_db

db = SessionLocal()

# Save a new scorecard
scorecard = save_scorecard_to_db(
    db=db,
    username="Colin",
    course_name="Championship Course",
    scores=[4, 4, 5, 3, 4, 5, 3, 4, 4, 4, 4, 4, 3, 4, 4, 3, 4, 5],
    course_pars=[4, 4, 4, 3, 4, 5, 3, 4, 4, 4, 4, 4, 3, 4, 4, 3, 4, 5],
    image_path="scorecard_photo.jpg"
)

print(f"Scorecard saved: {scorecard.id}")
```

## Database Operations in Python

```python
from database import SessionLocal
from models import User, Course, Scorecard, Score

db = SessionLocal()

# Get user with all scorecards
user = db.query(User).filter(User.username == "Colin").first()
print(f"User: {user.username}")
print(f"Total rounds: {len(user.scorecards)}")

# Get user's average score
scores = [sc.get_total_score() for sc in user.scorecards]
print(f"Average: {sum(scores)/len(scores):.1f}")

# Get specific scorecard details
scorecard = user.scorecards[0]
print(f"Course: {scorecard.course.name}")
print(f"Score: {scorecard.get_total_score()}")
print(f"Differential: {scorecard.get_score_differential():+d}")
print(f"Breakdown: {scorecard.get_hole_breakdown()}")

db.close()
```

## File Locations

| Item | Location |
|------|----------|
| Database | `scorecard.db` |
| Scorecard Images | `images/users/{username}/scorecards/` |
| Configuration | `database.py` |
| Models | `models.py` |
| API Server | `api.py` |
| Analysis Tool | `BreakDown.py` |
| Documentation | `DATABASE.md` |
| Original Data | `ScoreCard.csv` |

## Supported Backend Databases

### Current: SQLite
- File-based
- No server setup
- Perfect for single-user/small teams

### Alternative Options
- **PostgreSQL** - Better for multi-user concurrent access
- **MongoDB** - For flexible/evolving schemas (not recommended for this use case)

## Troubleshooting

### "Database is locked"
- Only one process can write at a time in SQLite
- Wait for other operations to complete
- Consider PostgreSQL for higher concurrency

### "User not found"
- Check username spelling/capitalization
- Query `/users` endpoint to see available users

### API port 8000 already in use
```bash
lsof -i :8000
kill -9 <PID>
```

## Next Steps

1. Integrate OCR pipeline with database saves
2. Add web UI dashboard
3. Set up automated backups
4. Add more courses to database
5. Track progression trends over time

See `DATABASE.md` for full documentation.
