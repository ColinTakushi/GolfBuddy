"""Utilities for saving OCR results to the database."""
import os
import shutil
from datetime import datetime
from pathlib import Path
from sqlalchemy.orm import Session
from models import User, Course, Scorecard, Score


def create_image_storage_dir(username: str) -> str:
    """Create and return the image storage directory for a user."""
    images_dir = os.path.join(os.path.dirname(__file__), "images", "users", username, "scorecards")
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
        raw_ocr_data: Optional raw OCR data for debugging
    
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
        # Copy image to user's storage directory
        images_dir = create_image_storage_dir(username)
        timestamp = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        filename = f"scorecard_{timestamp}.jpg"
        dest_path = os.path.join(images_dir, filename)
        shutil.copy2(image_path, dest_path)
        stored_image_path = dest_path
    
    # Create scorecard
    scorecard = Scorecard(
        user_id=user.id,
        course_id=course.id,
        date=datetime.utcnow(),
        image_path=stored_image_path,
        raw_ocr_data=raw_ocr_data
    )
    db.add(scorecard)
    db.flush()
    
    # Add individual hole scores
    for hole_num, score in enumerate(scores, start=1):
        score_obj = Score(
            scorecard_id=scorecard.id,
            hole_number=hole_num,
            score=score
        )
        db.add(score_obj)
    
    db.commit()
    db.refresh(scorecard)
    
    return scorecard


def store_ocr_result(
    ocr_result: list,
    output_csv_path: str = None
) -> dict:
    """
    Parse OCR result and extract structured data.
    
    Args:
        ocr_result: EasyOCR output (list of tuples with (bbox, text, confidence))
        output_csv_path: Optional path to save raw OCR data to CSV
    
    Returns:
        Dictionary with parsed OCR data
    """
    parsed_data = {
        "raw_detections": [],
        "text_by_confidence": [],
        "average_confidence": 0
    }
    
    total_confidence = 0
    count = 0
    
    for detection in ocr_result:
        bbox, text, confidence = detection
        total_confidence += confidence
        count += 1
        
        # Store raw detection
        parsed_data["raw_detections"].append({
            "bbox": bbox,
            "text": text,
            "confidence": confidence
        })
        
        # Store text by confidence (highest first)
        parsed_data["text_by_confidence"].append({
            "text": text,
            "confidence": confidence
        })
    
    if count > 0:
        parsed_data["average_confidence"] = total_confidence / count
    
    # Sort by confidence descending
    parsed_data["text_by_confidence"].sort(key=lambda x: x["confidence"], reverse=True)
    
    # Save to CSV if requested
    if output_csv_path:
        import csv
        with open(output_csv_path, 'w', newline='') as csvfile:
            writer = csv.DictWriter(csvfile, fieldnames=['bbox', 'text', 'conf'])
            writer.writeheader()
            for detection in parsed_data["raw_detections"]:
                writer.writerow({
                    'bbox': detection['bbox'],
                    'text': detection['text'],
                    'conf': detection['confidence']
                })
    
    return parsed_data
