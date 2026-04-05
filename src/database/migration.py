"""Migration script to import existing CSV data into SQLite database."""
import pandas as pd
import os
from datetime import datetime, timedelta
from src.core.db import SessionLocal, init_db
from src.core.models import User, Course, Scorecard


def migrate_from_csv():
    """Import data from ScoreCard.csv into the database."""
    init_db()
    db = SessionLocal()

    try:
        csv_path = os.path.join(os.path.dirname(__file__), "..", "..", "ScoreCard.csv")
        df = pd.read_csv(csv_path)

        print(f"Reading CSV from {csv_path}")
        print(f"Columns: {df.columns.tolist()}")
        print(f"Shape: {df.shape}")

        par_values = None
        for idx, row in df.iterrows():
            if row.iloc[0] == "Par":
                par_values = row[1:].astype(int).tolist()
                print(f"\nPar values extracted: {par_values}")
                break

        if par_values is None:
            raise ValueError("No 'Par' row found in CSV")

        course_name = "Championship Course"
        course = db.query(Course).filter(Course.name == course_name).first()
        if not course:
            course = Course(
                name=course_name,
                hole_pars=par_values,
                total_par=sum(par_values),
            )
            db.add(course)
            db.flush()
            print(f"Created course: {course_name}")
        else:
            print(f"Course {course_name} already exists")

        base_date = datetime(2024, 1, 1)
        player_count = 0

        for idx, row in df.iterrows():
            player_name = row.iloc[0]
            if player_name == "Par" or pd.isna(player_name) or str(player_name).strip() == "":
                continue

            print(f"\nProcessing player: {player_name}")

            user = db.query(User).filter(User.username == player_name).first()
            if not user:
                user = User(username=player_name)
                db.add(user)
                db.flush()
                print(f"  Created user: {player_name}")
            else:
                print(f"  User {player_name} already exists")

            scores_list = row[1:].astype(int).tolist()
            scorecard_date = base_date + timedelta(days=player_count)

            scorecard = Scorecard(
                course_id=course.id,
                date_played=scorecard_date,
                scores={str(user.id): scores_list},
                hole_layout=par_values,
                total_par=sum(par_values),
            )
            db.add(scorecard)
            db.flush()
            scorecard.users.append(user)

            db.commit()
            print(f"  Created scorecard for {scorecard_date.date()} with 18 scores")
            player_count += 1

        print("\n Migration completed successfully!")

    except Exception as e:
        db.rollback()
        print(f" Migration failed: {e}")
        import traceback
        traceback.print_exc()
    finally:
        db.close()


if __name__ == "__main__":
    migrate_from_csv()
