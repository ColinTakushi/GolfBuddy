"""Utilities for saving scorecard data to the database."""
from datetime import datetime
from sqlalchemy.orm import Session
from src.core.models import User, Course, Scorecard, scorecard_players


def save_scorecard_to_db(
    db: Session,
    course_name: str,
    course_pars: list,
    players: list,
    image_path: str = None,
    raw_ocr_data: dict = None,
) -> Scorecard:
    """
    Save a round (all players) as a single Scorecard.

    Args:
        db: SQLAlchemy session
        course_name: Name of the golf course
        course_pars: List of 18 par values
        players: List of dicts with 'name' and 'scores' keys
        image_path: Optional path to the scorecard image file
        raw_ocr_data: Optional raw OCR output for debugging

    Returns:
        The created Scorecard object
    """
    if len(course_pars) != 18:
        raise ValueError("Must provide exactly 18 par values")
    for p in players:
        if len(p["scores"]) != 18:
            raise ValueError(f"Player '{p['name']}': must provide exactly 18 scores")

    # Create or get course
    course = db.query(Course).filter(Course.name == course_name).first()
    if not course:
        course = Course(
            name=course_name,
            hole_pars=course_pars,
            total_par=sum(course_pars),
        )
        db.add(course)
        db.flush()

    # Create or get each user, collect (user, scores) pairs
    user_score_pairs = []
    for player in players:
        user = db.query(User).filter(User.username == player["name"]).first()
        if not user:
            user = User(username=player["name"])
            db.add(user)
            db.flush()
        user_score_pairs.append((user, player["scores"]))

    scores_json = {str(u.id): s for u, s in user_score_pairs}

    scorecard = Scorecard(
        course_id=course.id,
        image_path=image_path,
        raw_ocr_data=raw_ocr_data,
        date_played=datetime.now(),
        scores=scores_json,
        hole_layout=course_pars,
        total_par=sum(course_pars),
    )
    db.add(scorecard)
    db.flush()

    for user, _ in user_score_pairs:
        scorecard.users.append(user)

    db.commit()
    db.refresh(scorecard)

    return scorecard
