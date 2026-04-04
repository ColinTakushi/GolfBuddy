"""Configuration settings for scorecard tracking system."""
import os
from pathlib import Path

# Project root directory
PROJECT_ROOT = Path(__file__).parent.parent

# Database configuration
DATABASE_DIR = PROJECT_ROOT / "data" / "db"
DATABASE_DIR.mkdir(parents=True, exist_ok=True)
DATABASE_URL = f"sqlite:///{DATABASE_DIR / 'scorecard.db'}"

# Data directories
RAW_DATA_DIR = PROJECT_ROOT / "data" / "raw"
RAW_DATA_DIR.mkdir(parents=True, exist_ok=True)

IMAGES_DIR = PROJECT_ROOT / "data" / "images"
IMAGES_DIR.mkdir(parents=True, exist_ok=True)

# CSV file locations
SCORECARD_CSV = RAW_DATA_DIR / "ScoreCard.csv"

# Logging
LOG_DIR = PROJECT_ROOT / "logs"
LOG_DIR.mkdir(exist_ok=True)
