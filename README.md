# Golf Scorecard Reader

A Python application that reads golf scorecards from images using OCR, stores round data in a SQLite database, and exposes a REST API for querying stats.

## Features

- OCR processing of scorecard images using EasyOCR and OpenCV (GPU-accelerated if available)
- SQLite database for storing users, courses, scorecards, and hole-by-hole scores
- REST API built with FastAPI
- CLI tools for stats and data migration

## Project Structure

```
score-card-reader/
├── src/
│   ├── core/           # Database engine and SQLAlchemy models
│   ├── api/            # FastAPI application and endpoints
│   ├── ocr/            # Image preprocessing and OCR logic
│   ├── database/       # CSV migration utilities
│   └── analysis/       # Statistics and analytics
├── tools/
│   └── analytics.py    # CLI analytics tool
├── analyze.py          # Run stats from the command line
├── server.py           # Start the API server
├── run_migration.py    # Migrate CSV data to the database
├── main.py             # Unified entry point
└── requirements.txt
```

## Setup

### Prerequisites

- Python 3.12+
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) installed on your system
- NVIDIA GPU (optional, for faster OCR)

### Install dependencies

```bash
python -m venv score_card_env
source score_card_env/bin/activate  # Windows: score_card_env\Scripts\activate
pip install -r requirements.txt
```

## Usage

### Start the API server

```bash
python server.py           # runs on port 8000
python server.py 8080      # custom port
```

API docs available at `http://localhost:8000/docs`

### View player stats

```bash
python analyze.py Colin        # stats for a specific user
python analyze.py              # list all users
```

### Migrate CSV data to the database

```bash
python run_migration.py
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/users` | List all users |
| GET | `/courses` | List all courses |
| GET | `/scorecards` | List scorecards |
| GET | `/stats` | Aggregated statistics |

Full docs available via Swagger UI at `/docs` when the server is running.

## Tech Stack

- **OCR**: EasyOCR, OpenCV, Pytesseract
- **API**: FastAPI, Uvicorn
- **Database**: SQLite via SQLAlchemy
- **Data**: Pandas, NumPy
