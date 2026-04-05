"""SQLAlchemy ORM models for scorecard tracking system."""
from sqlalchemy import Column, Integer, String, DateTime, ForeignKey, JSON, Table
from sqlalchemy.orm import relationship
from datetime import datetime
from src.core.db import Base


# Join table: one scorecard has many players, one player has many scorecards
scorecard_players = Table(
    "scorecard_players",
    Base.metadata,
    Column("scorecard_id", Integer, ForeignKey("scorecards.id"), primary_key=True),
    Column("user_id", Integer, ForeignKey("users.id"), primary_key=True),
)


class User(Base):
    """User model for tracking individual golfers."""
    __tablename__ = "users"

    id = Column(Integer, primary_key=True, index=True)
    username = Column(String, unique=True, index=True, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)

    scorecards = relationship("Scorecard", secondary=scorecard_players, back_populates="users")

    def __repr__(self):
        return f"<User(id={self.id}, username='{self.username}')>"


class Course(Base):
    """Golf course model storing par values for each hole."""
    __tablename__ = "courses"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, unique=True, index=True, nullable=False)
    hole_pars = Column(JSON, nullable=False)   # [int×18]
    total_par = Column(Integer, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)

    scorecards = relationship("Scorecard", back_populates="course")

    def get_front_9_par(self):
        return sum(self.hole_pars[:9]) if self.hole_pars else 0

    def get_back_9_par(self):
        return sum(self.hole_pars[9:]) if self.hole_pars else 0

    def __repr__(self):
        return f"<Course(id={self.id}, name='{self.name}')>"


class Scorecard(Base):
    """One scorecard per round, shared across all players who participated."""
    __tablename__ = "scorecards"

    id = Column(Integer, primary_key=True, index=True)
    course_id = Column(Integer, ForeignKey("courses.id"), nullable=False, index=True)
    image_path = Column(String, nullable=True)
    raw_ocr_data = Column(JSON, nullable=True)
    date_played = Column(DateTime, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    scores = Column(JSON, nullable=False)       # {"user_id": [int×18], ...}
    hole_layout = Column(JSON, nullable=False)  # [int×18] — pars at time of play
    total_par = Column(Integer, nullable=False)

    course = relationship("Course", back_populates="scorecards")
    users = relationship("User", secondary=scorecard_players, back_populates="scorecards")

    def get_total_score(self, user_id: int) -> int:
        return sum(self.scores.get(str(user_id), []))

    def get_front_9_score(self, user_id: int) -> int:
        return sum(self.scores.get(str(user_id), [])[:9])

    def get_back_9_score(self, user_id: int) -> int:
        return sum(self.scores.get(str(user_id), [])[9:])

    def get_score_differential(self, user_id: int) -> int:
        return self.get_total_score(user_id) - self.total_par

    def get_hole_breakdown(self, user_id: int) -> dict:
        breakdown = {"birdies": 0, "pars": 0, "bogeys": 0, "doubles_plus": 0}
        for score, par in zip(self.scores.get(str(user_id), []), self.hole_layout):
            diff = score - par
            if diff <= -1:
                breakdown["birdies"] += 1
            elif diff == 0:
                breakdown["pars"] += 1
            elif diff == 1:
                breakdown["bogeys"] += 1
            else:
                breakdown["doubles_plus"] += 1
        return breakdown

    def __repr__(self):
        return f"<Scorecard(id={self.id}, course_id={self.course_id}, date_played={self.date_played})>"
