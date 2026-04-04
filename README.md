# Golf Scorecard Reader

A Python application that reads golf scorecards from images using Gemini 2.5 Flash, stores round data in a SQLite database, and exposes a REST API for querying stats.

## Features

- Scorecard image reading via Gemini 2.5 Flash API (multi-player support)
- SQLite database for storing users, courses, scorecards, and hole-by-hole scores
- REST API built with FastAPI
- CLI tools for stats

## Project Structure

```
score-card-reader/
├── src/
│   ├── core/           # Database engine and SQLAlchemy models
│   ├── api/            # FastAPI application and endpoints
│   ├── ocr/            # Gemini pipeline and DB save utilities
│   └── analysis/       # Statistics and analytics
├── tools/
│   └── analytics.py    # CLI analytics tool
├── scan.py             # Scan a scorecard image (main entry point)
├── analyze.py          # Run stats from the command line
├── server.py           # Start the API server
└── requirements.txt
```

## Setup

### Prerequisites

- Python 3.12+
- A Gemini API key — set it as `GEMINI_API_KEY` in your environment

### Install dependencies

```bash
python3 -m venv score_card_env
source score_card_env/bin/activate  # Windows: score_card_env\Scripts\activate
pip install -r requirements.txt
```

## Usage

### TUI (recommended)

#### Install Go

**Linux (WSL/Ubuntu):**
```bash
sudo apt update && sudo apt install -y golang-go
```

**Or install the latest version manually:**
```bash
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
source ~/.zshrc
```

#### Run

From the project root:

```bash
cd tui && go mod tidy && cd ..   # first time only
go run ./tui
```

Navigate with ↑/↓, select with Enter, go back with Esc, quit with q.

---

### CLI (direct)

### Scan a scorecard image

```bash
GEMINI_API_KEY=<your_key> python3 scan.py golf.jpg
```

The pipeline sends the image to Gemini 2.5 Flash, displays the extracted course and player data as JSON, then walks you through confirming or correcting each section before saving to the database. One scorecard record is saved per player.

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

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/users` | List all users |
| GET | `/courses` | List all courses |
| GET | `/scorecards/{username}` | List scorecards for a user |
| GET | `/stats/{username}` | Aggregated statistics |

Full docs available via Swagger UI at `/docs` when the server is running.

## Tech Stack

- **Image reading**: Gemini 2.5 Flash (`google-genai`)
- **API**: FastAPI, Uvicorn
- **Database**: SQLite via SQLAlchemy
- **Data**: Pandas, NumPy
