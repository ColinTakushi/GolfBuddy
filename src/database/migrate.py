"""Migration script to import existing CSV data into SQLite database."""
import pandas as pd
import os
from datetime import datetime, timedelta
from src.database.db import SessionLocal, init_db
from src.database.models import User, Course, Scorecard, Score
from src.config import SCORECARD_CSV


def migrate_from_csv(csv_path=None):
    """Import data from ScoreCard.csv into the database."""
    
    if csv_path is None:
        csv_path = SCORECARD_CSV
    
    # Initialize database
    init_db()
    db = SessionLocal()
    
    try:
        # Read CSV file
        df = pd.read_csv(csv_path)
        
        print(f"Reading CSV from {csv_path}")
        print(f"Columns: {df.columns.tolist()}")
        print(f"Shape: {df.shape}")
        
        # Find the "Par" row - it contains the par values
        par_row_idx = None
        par_values = None
        
        for idx, row in df.iterrows():
            if row.iloc[0] == "Par":
                par_row_idx = idx
                par_values = row[1:].astype(int).tolist()
                print(f"\nPar values extracted: {par_values}")
                break
        
        if par_values is None:
            raise ValueError("No 'Par' row found in CSV")
        
        # Create course with the par values
        course_name = "Championship Course"  # Default name for the course
        course = db.query(Course).filter(Course.name == course_name).first()
        
        if not course:
            course = Course(
                name=course_name,
                holes_par=par_values
            )
            db.add(course)
            db.flush()
            print(f"Created course: {course_name}")
        else:
            print(f"Course {course_name} already exists")
        
        # Process player rows (skip Par row)
        base_date = datetime(2024, 1, 1)
        player_count = 0
        
        for idx, row in df.iterrows():
            player_name = row.iloc[0]
            
            # Skip Par row and empty rows
            if player_name == "Par" or pd.isna(player_name) or player_name.strip() == "":
                continue
            
            print(f"\nProcessing player: {player_name}")
            
            # Find or create user
            user = db.query(User).filter(User.username == player_name).first()
            if not user:
                user = User(username=player_name)
                db.add(user)
                db.flush()
                print(f"  Created user: {player_name}")
            else:
                print(f"  User {player_name} already exists")
            
            # Create a scorecard for this player
            # Use sequential dates for demo purposes
            scorecard_date = base_date + timedelta(days=player_count)
            
            scorecard = Scorecard(
                user_id=user.id,
                course_id=course.id,
                date=scorecard_date,
                image_path=None,
                raw_ocr_data=None
            )
            db.add(scorecard)
            db.flush()
            print(f"  Created scorecard for {scorecard_date.date()}")
            
            # Add individual hole scores
            scores_data = row[1:].astype(int).tolist()  # Skip first column (name)
            
            for hole_num, score in enumerate(scores_data, start=1):
                score_obj = Score(
                    scorecard_id=scorecard.id,
                    hole_number=hole_num,
                    score=score
                )
                db.add(score_obj)
            
            db.commit()
            print(f"  Added 18 hole scores")
            player_count += 1
        
        print("\n✓ Migration completed successfully!")
        
    except Exception as e:
        db.rollback()
        print(f"✗ Migration failed: {e}")
        import traceback
        traceback.print_exc()
    finally:
        db.close()


if __name__ == "__main__":
    migrate_from_csv()
