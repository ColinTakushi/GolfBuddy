"""FastAPI backend for scorecard tracking system."""
from fastapi import FastAPI, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from sqlalchemy import desc
from datetime import datetime, timedelta
from typing import List, Optional
from pydantic import BaseModel
from contextlib import asynccontextmanager

from src.database.db import init_db, get_db
from src.database.models import User, Course, Scorecard, scorecard_players

init_db()

app = FastAPI(title="Scorecard Tracking API")


# ============ Pydantic Models ============

class ScorecardPlayerBody(BaseModel):
    name: str
    scores: List[int]


class ScorecardCourseBody(BaseModel):
    name: str
    holePars: List[int]


class ScorecardBody(BaseModel):
    course: ScorecardCourseBody
    players: List[ScorecardPlayerBody]
    imagePath: Optional[str] = None


# ============ Root ============

@app.get("/")
async def root():
    return {"message": "Scorecard Tracking API", "version": "2.0"}


# ============ Admin ============

@app.delete("/nuke")
async def nuke_database(db: Session = Depends(get_db)):
    """Delete all data from all tables."""
    db.execute(scorecard_players.delete())
    db.query(Scorecard).delete()
    db.query(User).delete()
    db.query(Course).delete()
    db.commit()
    return {"message": "All data deleted."}


# ============ Users ============

@app.post("/users")
async def create_user(username: str, db: Session = Depends(get_db)):
    """Create a new user."""
    if db.query(User).filter(User.username == username).first():
        raise HTTPException(status_code=400, detail="User already exists")
    user = User(username=username)
    db.add(user)
    db.commit()
    db.refresh(user)
    return {"id": user.id, "username": user.username, "created_at": user.created_at}


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
    """Get user details and aggregate stats."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    scorecards = user.scorecards
    total_rounds = len(scorecards)

    if scorecards:
        scores = [sc.get_total_score(user.id) for sc in scorecards]
        average_score = sum(scores) / total_rounds
        best_score = min(scores)
        worst_score = max(scores)
        courses_played = len(set(sc.course_id for sc in scorecards))
    else:
        average_score = best_score = worst_score = None
        courses_played = 0

    return {
        "id": user.id,
        "username": user.username,
        "created_at": user.created_at,
        "total_rounds": total_rounds,
        "average_score": average_score,
        "best_score": best_score,
        "worst_score": worst_score,
        "courses_played": courses_played,
    }


@app.get("/{username}/scorecards")
async def get_user_scorecards(username: str, db: Session = Depends(get_db)):
    """Get all scorecards for a user, sorted newest first.

    Returns summary info for each round: course, date, score vs par,
    front/back 9 splits, hole breakdown, image path, and co-players.
    """
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    scorecards = sorted(
        user.scorecards, key=lambda sc: sc.date_played, reverse=True)

    return [
        {
            "id": sc.id,
            "course": sc.course.name,
            "date_played": sc.date_played,
            "total_score": sc.get_total_score(user.id),
            "total_par": sc.total_par,
            "score_differential": sc.get_score_differential(user.id),
            "front_9": sc.get_front_9_score(user.id),
            "back_9": sc.get_back_9_score(user.id),
            "hole_breakdown": sc.get_hole_breakdown(user.id),
            "image_path": sc.image_path,
            "players": [u.username for u in sc.users],
        }
        for sc in scorecards
    ]

# ============ Courses ============


@app.post("/courses")
async def create_course(name: str, holes_par: List[int], db: Session = Depends(get_db)):
    """Create a new course (exactly 18 par values required)."""
    if len(holes_par) != 18:
        raise HTTPException(
            status_code=400, detail="Must provide exactly 18 par values")
    if db.query(Course).filter(Course.name == name).first():
        raise HTTPException(status_code=400, detail="Course already exists")

    course = Course(name=name, hole_pars=holes_par, total_par=sum(holes_par))
    db.add(course)
    db.commit()
    db.refresh(course)

    return {
        "id": course.id,
        "name": course.name,
        "hole_pars": course.hole_pars,
        "total_par": course.total_par,
        "front_9_par": course.get_front_9_par(),
        "back_9_par": course.get_back_9_par(),
    }


@app.get("/courses")
async def list_courses(db: Session = Depends(get_db)):
    """List all courses."""
    courses = db.query(Course).all()
    return [
        {
            "id": c.id,
            "name": c.name,
            "total_par": c.total_par,
            "rounds_played": len(c.scorecards),
        }
        for c in courses
    ]


@app.get("/courses/{course_id}")
async def get_course(course_id: int, db: Session = Depends(get_db)):
    """Get course detail including average score across all rounds."""
    course = db.query(Course).filter(Course.id == course_id).first()
    if not course:
        raise HTTPException(status_code=404, detail="Course not found")

    all_scores = [
        sum(player_scores)
        for sc in course.scorecards
        for player_scores in sc.scores.values()
    ]
    average_score = sum(all_scores) / len(all_scores) if all_scores else None

    return {
        "id": course.id,
        "name": course.name,
        "hole_pars": course.hole_pars,
        "total_par": course.total_par,
        "front_9_par": course.get_front_9_par(),
        "back_9_par": course.get_back_9_par(),
        "rounds_played": len(course.scorecards),
        "average_score": average_score,
    }


# ============ Scorecards ============

@app.post("/scorecards")
async def create_scorecard(body: ScorecardBody, db: Session = Depends(get_db)):
    """Create one scorecard for a full round (all players)."""
    if len(body.course.holePars) != 18:
        raise HTTPException(
            status_code=400, detail="Must provide exactly 18 par values")
    for p in body.players:
        if len(p.scores) != 18:
            raise HTTPException(
                status_code=400, detail=f"Player {p.name}: must provide exactly 18 scores")

    # Find or create course
    course = db.query(Course).filter(Course.name == body.course.name).first()
    if not course:
        course = Course(name=body.course.name, hole_pars=body.course.holePars,
                        total_par=sum(body.course.holePars))
        db.add(course)
        db.flush()

    # Find or create each user, build scores dict
    user_objects = []
    for p in body.players:
        user = db.query(User).filter(User.username == p.name).first()
        if not user:
            user = User(username=p.name)
            db.add(user)
            db.flush()
        user_objects.append((user, p.scores))

    scores_json = {str(u.id): s for u, s in user_objects}

    scorecard = Scorecard(
        course_id=course.id,
        image_path=body.imagePath,
        date_played=datetime.now(),
        scores=scores_json,
        hole_layout=body.course.holePars,
        total_par=sum(body.course.holePars),
    )
    db.add(scorecard)
    db.flush()

    for user, _ in user_objects:
        scorecard.users.append(user)

    db.commit()
    db.refresh(scorecard)

    return {
        "id": scorecard.id,
        "course": course.name,
        "date_played": scorecard.date_played,
        "total_par": scorecard.total_par,
        "players": [
            {"username": u.username,
                "total_score": scorecard.get_total_score(u.id)}
            for u, _ in user_objects
        ],
    }


@app.delete("/scorecards/{scorecard_id}/{user_id}")
async def delete_scorecard_user(scorecard_id: int, user_id: int, db: Session = Depends(get_db)):
    """Remove a user from a scorecard.

    Removes the user from the scorecard's player list and deletes their scores.
    If no players remain on the scorecard after removal, the scorecard itself is deleted.
    """
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(status_code=404, detail="User Id not found")

    scorecard = db.query(Scorecard).filter(
        Scorecard.id == scorecard_id).first()
    if not scorecard or user not in scorecard.users:
        raise HTTPException(status_code=404, detail="Scorecard not found")

    scorecard.users.remove(user)

    updated_scores = dict(scorecard.scores)
    updated_scores.pop(str(user_id), None)
    scorecard.scores = updated_scores

    if not scorecard.users:
        db.delete(scorecard)

    db.commit()
    return {"message": "Scorecard removed from user history."}


@app.get("/scorecards/{username}/{scorecard_id}")
async def get_scorecard_detail(username: str, scorecard_id: int, db: Session = Depends(get_db)):
    """Get detailed scorecard with per-hole breakdown for the specified user."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    scorecard = db.query(Scorecard).filter(
        Scorecard.id == scorecard_id).first()
    if not scorecard or user not in scorecard.users:
        raise HTTPException(status_code=404, detail="Scorecard not found")

    user_scores = scorecard.scores.get(str(user.id), [])

    return {
        "scorecardId": scorecard.id,
        "user": user.username,
        "userId": user.id,
        "course": scorecard.course.name,
        "date_played": scorecard.date_played,
        "total_score": scorecard.get_total_score(user.id),
        "total_par": scorecard.total_par,
        "score_differential": scorecard.get_score_differential(user.id),
        "image_path": scorecard.image_path,
        "holes": [
            {
                "hole_number": i + 1,
                "score": score,
                "par": scorecard.hole_layout[i],
                "differential": score - scorecard.hole_layout[i],
            }
            for i, score in enumerate(user_scores)
        ],
        "front_9": {
            "score": scorecard.get_front_9_score(user.id),
            "par": scorecard.course.get_front_9_par(),
        },
        "back_9": {
            "score": scorecard.get_back_9_score(user.id),
            "par": scorecard.course.get_back_9_par(),
        },
        "hole_breakdown": scorecard.get_hole_breakdown(user.id),
        "players": [u.username for u in scorecard.users],
    }


@app.get("/rounds")
async def list_all_rounds(db: Session = Depends(get_db)):
    """List all rounds across all users, sorted newest first."""
    scorecards = db.query(Scorecard).order_by(desc(Scorecard.date_played)).all()
    result = []
    for sc in scorecards:
        for user in sc.users:
            result.append({
                "id": sc.id,
                "course": sc.course.name,
                "date_played": sc.date_played.isoformat() if sc.date_played else "",
                "total_score": sc.get_total_score(user.id),
                "total_par": sc.total_par,
                "score_differential": sc.get_score_differential(user.id),
                "player": user.username,
            })
    return result


@app.get("/scorecards/{scorecard_id}")
async def get_score_card_from_id(scorecard_id: int, db: Session = Depends(get_db)):
    """Get a scorecard by its ID."""

    sc = db.query(Scorecard).filter(Scorecard.id == scorecard_id).first()
    if not sc:
        raise HTTPException(status_code=404, detail="Score card not found")

    return {
        "sc": sc
    }


@app.put("/scorecards/{scorecard_id}")
async def update_scorecard_scores(
    scorecard_id: int,
    username: str,
    scores: List[int],
    db: Session = Depends(get_db),
):
    """Replace a player's 18 hole scores for an existing round."""
    if len(scores) != 18:
        raise HTTPException(
            status_code=400, detail="Must provide exactly 18 scores")

    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    sc = db.query(Scorecard).filter(Scorecard.id == scorecard_id).first()
    if not sc or user not in sc.users:
        raise HTTPException(status_code=404, detail="Scorecard not found")

    updated = dict(sc.scores)
    updated[str(user.id)] = scores
    sc.scores = updated
    db.commit()
    db.refresh(sc)

    return {"id": sc.id, "username": username, "total_score": sc.get_total_score(user.id)}


# ============ Statistics ============

@app.get("/stats/{username}")
async def get_user_stats(
    username: str,
    days: Optional[int] = Query(None, description="Filter to last N days"),
    db: Session = Depends(get_db),
):
    """Get aggregated statistics for a user."""
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    scorecards = user.scorecards
    if days:
        cutoff = datetime.now() - timedelta(days=days)
        scorecards = [sc for sc in scorecards if sc.date_played >= cutoff]

    scorecards = sorted(scorecards, key=lambda sc: sc.date_played)

    if not scorecards:
        raise HTTPException(
            status_code=404, detail="No scorecards found for this user")

    scores = [sc.get_total_score(user.id) for sc in scorecards]

    course_stats = {}
    for sc in scorecards:
        cname = sc.course.name
        if cname not in course_stats:
            course_stats[cname] = {
                "rounds": 0, "total_score": 0, "best": None, "worst": None}
        s = sc.get_total_score(user.id)
        course_stats[cname]["rounds"] += 1
        course_stats[cname]["total_score"] += s
        course_stats[cname]["best"] = min(s, course_stats[cname]["best"] or s)
        course_stats[cname]["worst"] = max(
            s, course_stats[cname]["worst"] or s)

    for cname in course_stats:
        course_stats[cname]["average"] = course_stats[cname]["total_score"] / \
            course_stats[cname]["rounds"]

    return {
        "username": username,
        "total_rounds": len(scorecards),
        "average_score": sum(scores) / len(scores),
        "best_score": min(scores),
        "worst_score": max(scores),
        "scores_trend": scores,
        "course_breakdown": course_stats,
        "handicap_estimate": sum(scores) / len(scores) - 72,
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

    scorecards = sorted(
        [sc for sc in user.scorecards if sc.course_id == course_id],
        key=lambda sc: sc.date_played,
    )

    if not scorecards:
        raise HTTPException(
            status_code=404, detail="No scorecards found for this user at this course")

    scores = [sc.get_total_score(user.id) for sc in scorecards]
    differentials = [sc.get_score_differential(user.id) for sc in scorecards]

    return {
        "username": username,
        "course": course.name,
        "rounds": len(scorecards),
        "average_score": sum(scores) / len(scores),
        "best_score": min(scores),
        "worst_score": max(scores),
        "average_differential": sum(differentials) / len(differentials),
        "course_par": course.total_par,
        "scores": scores,
        "dates": [sc.date_played for sc in scorecards],
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
