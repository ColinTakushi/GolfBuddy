"""FastAPI backend for scorecard data access."""
from fastapi import FastAPI, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from sqlalchemy import func, desc
from datetime import datetime, timedelta
from typing import List, Optional
import os
from pydantic import BaseModel

from database import SessionLocal, init_db, get_db
from models import User, Course, Scorecard, Score

# Initialize database
init_db()

# Create FastAPI app
app = FastAPI(title="Scorecard Tracking API")


# ============ Pydantic Models (Response schemas) ============

class ScoreResponse(BaseModel):
    hole_number: int
    score: int
    
    class Config:
        from_attributes = True


class ScorecardResponse(BaseModel):
    id: int
    user_id: int
    course_id: int
    date: datetime
    image_path: Optional[str]
    scores: List[ScoreResponse]
    total_score: int
    total_par: int
    score_differential: int
    
    class Config:
        from_attributes = True
    
    @property
    def total_score(self) -> int:
        return sum(s.score for s in self.scores) if self.scores else 0
    
    @property
    def total_par(self) -> int:
        return 72  # Placeholder - should come from course
    
    @property
    def score_differential(self) -> int:
        return self.total_score - self.total_par


class CourseResponse(BaseModel):
    id: int
    name: str
    holes_par: List[int]
    
    class Config:
        from_attributes = True
    
    @property
    def total_par(self) -> int:
        return sum(self.holes_par)
    
    @property
    def front_9_par(self) -> int:
        return sum(self.holes_par[:9])
    
    @property
    def back_9_par(self) -> int:
        return sum(self.holes_par[9:])


class UserProgressionResponse(BaseModel):
    username: str
    total_rounds: int
    average_score: Optional[float]
    best_score: Optional[int]
    worst_score: Optional[int]
    courses_played: int
    
    class Config:
        from_attributes = True


class UserDetailResponse(BaseModel):
    id: int
    username: str
    created_at: datetime
    total_rounds: int
    average_score: Optional[float]
    
    class Config:
        from_attributes = True


# ============ Endpoints ============

@app.get("/")
async def root():
    """Root endpoint."""
    return {"message": "Scorecard Tracking API", "version": "1.0"}


# ============ Users ============

@app.post("/users")
async def create_user(username: str, db: Session = Depends(get_db)):
    """Create a new user."""
    # Check if user already exists
    existing_user = db.query(User).filter(User.username == username).first()
    if existing_user:
        raise HTTPException(status_code=400, detail="User already exists")
    
    user = User(username=username)
    db.add(user)
    db.commit()
    db.refresh(user)
    return {
        "id": user.id,
        "username": user.username,
        "created_at": user.created_at
    }


@app.get("/users")
async def list_users(db: Session = Depends(get_db)):
    """List all users."""
    users = db.query(User).all()
    return [
        {
            "id": u.id,
            "username": u.username,
            "scorecards_count": len(u.scorecards)
        }
        for u in users
    ]


@app.get("/users/{username}")
async def get_user(username: str, db: Session = Depends(get_db)):
    """Get user details and progression stats."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    
    # Calculate statistics
    scorecards = user.scorecards
    total_rounds = len(scorecards)
    
    if scorecards:
        total_score = sum(sc.get_total_score() for sc in scorecards)
        average_score = total_score / total_rounds
        best_score = min(sc.get_total_score() for sc in scorecards)
        worst_score = max(sc.get_total_score() for sc in scorecards)
        courses_played = len(set(sc.course_id for sc in scorecards))
    else:
        average_score = None
        best_score = None
        worst_score = None
        courses_played = 0
    
    return {
        "id": user.id,
        "username": user.username,
        "created_at": user.created_at,
        "total_rounds": total_rounds,
        "average_score": average_score,
        "best_score": best_score,
        "worst_score": worst_score,
        "courses_played": courses_played
    }


# ============ Courses ============

@app.post("/courses")
async def create_course(name: str, holes_par: List[int], db: Session = Depends(get_db)):
    """Create a new course (18 hole par values)."""
    if len(holes_par) != 18:
        raise HTTPException(status_code=400, detail="Must provide exactly 18 par values")
    
    existing_course = db.query(Course).filter(Course.name == name).first()
    if existing_course:
        raise HTTPException(status_code=400, detail="Course already exists")
    
    course = Course(name=name, holes_par=holes_par)
    db.add(course)
    db.commit()
    db.refresh(course)
    
    return {
        "id": course.id,
        "name": course.name,
        "holes_par": course.holes_par,
        "total_par": course.get_total_par(),
        "front_9_par": course.get_front_9_par(),
        "back_9_par": course.get_back_9_par()
    }


@app.get("/courses")
async def list_courses(db: Session = Depends(get_db)):
    """List all courses."""
    courses = db.query(Course).all()
    return [
        {
            "id": c.id,
            "name": c.name,
            "total_par": c.get_total_par(),
            "rounds_played": len(c.scorecards)
        }
        for c in courses
    ]


@app.get("/courses/{course_id}")
async def get_course(course_id: int, db: Session = Depends(get_db)):
    """Get course details and statistics."""
    course = db.query(Course).filter(Course.id == course_id).first()
    if not course:
        raise HTTPException(status_code=404, detail="Course not found")
    
    return {
        "id": course.id,
        "name": course.name,
        "holes_par": course.holes_par,
        "total_par": course.get_total_par(),
        "front_9_par": course.get_front_9_par(),
        "back_9_par": course.get_back_9_par(),
        "rounds_played": len(course.scorecards),
        "average_score": sum(sc.get_total_score() for sc in course.scorecards) / len(course.scorecards) if course.scorecards else None
    }


# ============ Scorecards ============

@app.post("/scorecards")
async def create_scorecard(
    username: str,
    course_id: int,
    scores: List[int],
    image_path: Optional[str] = None,
    db: Session = Depends(get_db)
):
    """Create a new scorecard for a user."""
    if len(scores) != 18:
        raise HTTPException(status_code=400, detail="Must provide exactly 18 scores")
    
    # Find user
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    
    # Find course
    course = db.query(Course).filter(Course.id == course_id).first()
    if not course:
        raise HTTPException(status_code=404, detail="Course not found")
    
    # Create scorecard
    scorecard = Scorecard(
        user_id=user.id,
        course_id=course.id,
        date=datetime.utcnow(),
        image_path=image_path
    )
    db.add(scorecard)
    db.flush()
    
    # Add individual scores
    for hole_num, score in enumerate(scores, start=1):
        score_obj = Score(
            scorecard_id=scorecard.id,
            hole_number=hole_num,
            score=score
        )
        db.add(score_obj)
    
    db.commit()
    db.refresh(scorecard)
    
    return {
        "id": scorecard.id,
        "user_id": scorecard.user_id,
        "course_id": scorecard.course_id,
        "date": scorecard.date,
        "total_score": scorecard.get_total_score(),
        "total_par": scorecard.get_total_par(),
        "score_differential": scorecard.get_score_differential(),
        "image_path": scorecard.image_path
    }


@app.get("/scorecards/{username}")
async def get_user_scorecards(username: str, db: Session = Depends(get_db)):
    """Get all scorecards for a user."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    
    scorecards = db.query(Scorecard).filter(Scorecard.user_id == user.id).order_by(desc(Scorecard.date)).all()
    
    return [
        {
            "id": sc.id,
            "course": sc.course.name,
            "date": sc.date,
            "total_score": sc.get_total_score(),
            "total_par": sc.get_total_par(),
            "score_differential": sc.get_score_differential(),
            "front_9": sc.get_front_9_score(),
            "back_9": sc.get_back_9_score(),
            "hole_breakdown": sc.get_hole_breakdown(),
            "image_path": sc.image_path
        }
        for sc in scorecards
    ]


@app.get("/scorecards/{username}/{scorecard_id}")
async def get_scorecard_detail(username: str, scorecard_id: int, db: Session = Depends(get_db)):
    """Get detailed scorecard with all hole scores."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    
    scorecard = db.query(Scorecard).filter(
        Scorecard.id == scorecard_id,
        Scorecard.user_id == user.id
    ).first()
    
    if not scorecard:
        raise HTTPException(status_code=404, detail="Scorecard not found")
    
    scores = sorted(scorecard.scores, key=lambda s: s.hole_number)
    course_pars = scorecard.course.holes_par
    
    return {
        "id": scorecard.id,
        "user": scorecard.user.username,
        "course": scorecard.course.name,
        "date": scorecard.date,
        "total_score": scorecard.get_total_score(),
        "total_par": scorecard.get_total_par(),
        "score_differential": scorecard.get_score_differential(),
        "image_path": scorecard.image_path,
        "holes": [
            {
                "hole_number": s.hole_number,
                "score": s.score,
                "par": course_pars[s.hole_number - 1],
                "differential": s.score - course_pars[s.hole_number - 1]
            }
            for s in scores
        ],
        "front_9": {
            "score": scorecard.get_front_9_score(),
            "par": scorecard.course.get_front_9_par()
        },
        "back_9": {
            "score": scorecard.get_back_9_score(),
            "par": scorecard.course.get_back_9_par()
        },
        "hole_breakdown": scorecard.get_hole_breakdown()
    }


# ============ Statistics ============

@app.get("/stats/{username}")
async def get_user_stats(
    username: str,
    days: Optional[int] = Query(None, description="Filter to last N days"),
    db: Session = Depends(get_db)
):
    """Get aggregated statistics for a user."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    
    scorecards = db.query(Scorecard).filter(Scorecard.user_id == user.id)
    
    # Filter by days if specified
    if days:
        cutoff_date = datetime.utcnow() - timedelta(days=days)
        scorecards = scorecards.filter(Scorecard.date >= cutoff_date)
    
    scorecards = scorecards.order_by(Scorecard.date).all()
    
    if not scorecards:
        raise HTTPException(status_code=404, detail="No scorecards found for this user")
    
    scores = [sc.get_total_score() for sc in scorecards]
    
    # Course breakdown
    course_stats = {}
    for sc in scorecards:
        course_name = sc.course.name
        if course_name not in course_stats:
            course_stats[course_name] = {
                "rounds": 0,
                "total_score": 0,
                "best": None,
                "worst": None
            }
        course_stats[course_name]["rounds"] += 1
        course_stat_score = sc.get_total_score()
        course_stats[course_name]["total_score"] += course_stat_score
        if course_stats[course_name]["best"] is None:
            course_stats[course_name]["best"] = course_stat_score
        else:
            course_stats[course_name]["best"] = min(course_stats[course_name]["best"], course_stat_score)
        if course_stats[course_name]["worst"] is None:
            course_stats[course_name]["worst"] = course_stat_score
        else:
            course_stats[course_name]["worst"] = max(course_stats[course_name]["worst"], course_stat_score)
    
    # Calculate averages for each course
    for course_name in course_stats:
        course_stats[course_name]["average"] = course_stats[course_name]["total_score"] / course_stats[course_name]["rounds"]
    
    return {
        "username": username,
        "total_rounds": len(scorecards),
        "average_score": sum(scores) / len(scores),
        "best_score": min(scores),
        "worst_score": max(scores),
        "scores_trend": scores,
        "course_breakdown": course_stats,
        "handicap_estimate": sum(scores) / len(scores) - 72  # Rough estimate
    }


@app.get("/stats/{username}/course/{course_id}")
async def get_user_course_stats(username: str, course_id: int, db: Session = Depends(get_db)):
    """Get statistics for a user at a specific course."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    
    course = db.query(Course).filter(Course.id == course_id).first()
    if not course:
        raise HTTPException(status_code=404, detail="Course not found")
    
    scorecards = db.query(Scorecard).filter(
        Scorecard.user_id == user.id,
        Scorecard.course_id == course_id
    ).order_by(Scorecard.date).all()
    
    if not scorecards:
        raise HTTPException(status_code=404, detail="No scorecards found for this user at this course")
    
    scores = [sc.get_total_score() for sc in scorecards]
    differentials = [sc.get_score_differential() for sc in scorecards]
    
    return {
        "username": username,
        "course": course.name,
        "rounds": len(scorecards),
        "average_score": sum(scores) / len(scores),
        "best_score": min(scores),
        "worst_score": max(scores),
        "average_differential": sum(differentials) / len(differentials),
        "course_par": course.get_total_par(),
        "scores": scores,
        "dates": [sc.date for sc in scorecards]
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
