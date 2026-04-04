"""Utilities for saving scorecard data to the database."""
import os
import shutil
from datetime import datetime
from pathlib import Path
from sqlalchemy.orm import Session
from src.core.models import User, Course, Scorecard, Score


def create_image_storage_dir(username: str) -> str:
    """Create and return the image storage directory for a user."""
    images_dir = os.path.join(os.path.dirname(__file__), "..", "..", "images", "users", username, "scorecards")
    Path(images_dir).mkdir(parents=True, exist_ok=True)
    return images_dir


def save_scorecard_to_db(
    db: Session,
    username: str,
    course_name: str,
    scores: list,
    course_pars: list,
    image_path: str = None,
    raw_ocr_data: dict = None
) -> Scorecard:
    """
    Save a scorecard to the database.

    Args:
        db: SQLAlchemy session
        username: Username of the golfer
        course_name: Name of the golf course
        scores: List of 18 scores (integers)
        course_pars: List of 18 par values (integers)
        image_path: Optional path to the scorecard image
        raw_ocr_data: Optional raw extracted data for debugging

    Returns:
        Scorecard object that was created
    """
    if len(scores) != 18:
        raise ValueError("Must provide exactly 18 scores")

    if len(course_pars) != 18:
        raise ValueError("Must provide exactly 18 par values")

    # Create or get user
    user = db.query(User).filter(User.username == username).first()
    if not user:
        user = User(username=username)
        db.add(user)
        db.flush()

    # Create or get course
    course = db.query(Course).filter(Course.name == course_name).first()
    if not course:
        course = Course(name=course_name, holes_par=course_pars)
        db.add(course)
        db.flush()

    # Process image if provided
    stored_image_path = None
    if image_path and os.path.exists(image_path):
        images_dir = create_image_storage_dir(username)
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"scorecard_{timestamp}.jpg"
        dest_path = os.path.join(images_dir, filename)
        shutil.copy2(image_path, dest_path)
        stored_image_path = dest_path

    # Create scorecard
    scorecard = Scorecard(
        user_id=user.id,
        course_id=course.id,
        date=datetime.now(),
        image_path=stored_image_path,
        raw_ocr_data=raw_ocr_data,
    )
    db.add(scorecard)
    db.flush()

    # Add individual hole scores
    for hole_num, score in enumerate(scores, start=1):
        score_obj = Score(
            scorecard_id=scorecard.id,
            hole_number=hole_num,
            score=score,
        )
        db.add(score_obj)

    db.commit()
    db.refresh(scorecard)

    return scorecard
