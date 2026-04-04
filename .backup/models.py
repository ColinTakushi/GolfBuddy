"""SQLAlchemy ORM models for scorecard tracking system."""
from sqlalchemy import Column, Integer, String, Float, DateTime, ForeignKey, JSON, Text
from sqlalchemy.orm import relationship
from datetime import datetime
import json
from database import Base


class User(Base):
    """User model for tracking individual golfers."""
    __tablename__ = "users"
    
    id = Column(Integer, primary_key=True, index=True)
    username = Column(String, unique=True, index=True, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relationships
    scorecards = relationship("Scorecard", back_populates="user", cascade="all, delete-orphan")
    
    def __repr__(self):
        return f"<User(id={self.id}, username='{self.username}')>"


class Course(Base):
    """Golf course model storing par values for each hole."""
    __tablename__ = "courses"
    
    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, unique=True, index=True, nullable=False)
    # Store par values as JSON array: [4, 4, 4, 3, 4, 5, 3, 4, 4, 4, 4, 4, 3, 4, 5, 3, 4, 4]
    holes_par = Column(JSON, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relationships
    scorecards = relationship("Scorecard", back_populates="course", cascade="all, delete-orphan")
    
    def get_total_par(self):
        """Get total par for 18 holes."""
        return sum(self.holes_par) if self.holes_par else 0
    
    def get_front_9_par(self):
        """Get par for front 9 (holes 1-9)."""
        return sum(self.holes_par[:9]) if self.holes_par else 0
    
    def get_back_9_par(self):
        """Get par for back 9 (holes 10-18)."""
        return sum(self.holes_par[9:]) if self.holes_par else 0
    
    def __repr__(self):
        return f"<Course(id={self.id}, name='{self.name}')>"


class Scorecard(Base):
    """Scorecard representing one round of golf."""
    __tablename__ = "scorecards"
    
    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)
    course_id = Column(Integer, ForeignKey("courses.id"), nullable=False, index=True)
    date = Column(DateTime, nullable=False)
    image_path = Column(String, nullable=True)  # Path to the scorecard image file
    raw_ocr_data = Column(JSON, nullable=True)  # Raw OCR output for debugging
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relationships
    user = relationship("User", back_populates="scorecards")
    course = relationship("Course", back_populates="scorecards")
    scores = relationship("Score", back_populates="scorecard", cascade="all, delete-orphan")
    
    def get_total_score(self):
        """Calculate total score for 18 holes."""
        return sum(score.score for score in self.scores) if self.scores else 0
    
    def get_total_par(self):
        """Get total par from the course."""
        return self.course.get_total_par()
    
    def get_score_differential(self):
        """Get score difference from par (negative is good)."""
        return self.get_total_score() - self.get_total_par()
    
    def get_front_9_score(self):
        """Get score for front 9."""
        return sum(s.score for s in self.scores if s.hole_number <= 9)
    
    def get_back_9_score(self):
        """Get score for back 9."""
        return sum(s.score for s in self.scores if s.hole_number > 9)
    
    def get_hole_breakdown(self):
        """Get breakdown of birdies, pars, bogeys, etc."""
        breakdown = {
            "birdies": 0,
            "pars": 0,
            "bogeys": 0,
            "doubles_plus": 0
        }
        for score in self.scores:
            par = self.course.holes_par[score.hole_number - 1]
            diff = score.score - par
            if diff == -1:
                breakdown["birdies"] += 1
            elif diff == 0:
                breakdown["pars"] += 1
            elif diff == 1:
                breakdown["bogeys"] += 1
            else:
                breakdown["doubles_plus"] += 1
        return breakdown
    
    def __repr__(self):
        return f"<Scorecard(id={self.id}, user_id={self.user_id}, course_id={self.course_id}, date={self.date})>"


class Score(Base):
    """Individual hole score."""
    __tablename__ = "scores"
    
    id = Column(Integer, primary_key=True, index=True)
    scorecard_id = Column(Integer, ForeignKey("scorecards.id"), nullable=False, index=True)
    hole_number = Column(Integer, nullable=False)  # 1-18
    score = Column(Integer, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relationships
    scorecard = relationship("Scorecard", back_populates="scores")
    
    __table_args__ = (
        # Ensure hole_number is between 1 and 18
    )
    
    def __repr__(self):
        return f"<Score(id={self.id}, hole={self.hole_number}, score={self.score})>"
