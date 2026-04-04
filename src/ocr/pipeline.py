"""Gemini-based pipeline: image → API → review → save → analyze."""
import csv
import io
import json
import mimetypes
import os
import re
import sys

from google import genai
from google.genai import types

from src.ocr.utils import save_scorecard_to_db
from src.core.db import SessionLocal
from tools.analytics import print_user_breakdown


_PROMPT = """
Analyze this golf scorecard image.
Extract the course name, the par for each hole in order, total par, front 9 par, and back 9 par.
Extract all player names, their per-hole scores, and totals.
Return ONLY valid JSON with no markdown fences in this exact format:
{
  "course": {
    "name": "Course Name",
    "holePars": [4, 3, 5, 4, 4, 4, 4, 3, 5, 4, 4, 4, 3, 4, 5, 4, 4, 4],
    "par": 72,
    "front": 36,
    "back": 36
  },
  "players": [
    { "name": "Player Name", "scores": [4, 3, 5, ...], "total": 75 }
  ]
}
"""


def _call_gemini(image_path: str) -> dict:
    """Send image to Gemini and return parsed JSON dict."""
    client = genai.Client()

    mime_type, _ = mimetypes.guess_type(image_path)
    if mime_type not in ("image/jpeg", "image/png", "image/webp", "image/heic", "image/heif"):
        mime_type = "image/jpeg"

    with open(image_path, "rb") as f:
        image_bytes = f.read()

    image_part = types.Part.from_bytes(data=image_bytes, mime_type=mime_type)

    print("Sending scorecard to Gemini 2.5 Flash...")
    response = client.models.generate_content(
        model="gemini-2.5-flash",
        contents=[_PROMPT, image_part],
    )
    raw = response.text.strip()
    # Strip markdown code fences if present
    raw = re.sub(r"^```(?:json)?\s*", "", raw)
    raw = re.sub(r"\s*```$", "", raw)
    return json.loads(raw)


def _write_scan_csv(image_path: str, player_name: str, scores: list, pars: list, course_name: str) -> str:
    """Write per-hole breakdown CSV next to the source image."""
    base = os.path.splitext(image_path)[0]
    safe_name = player_name.replace(" ", "_")
    csv_path = f"{base}_{safe_name}_scan_result.csv"
    with open(csv_path, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["hole", "par", "score", "vs_par"])
        for i, (par, score) in enumerate(zip(pars, scores), 1):
            writer.writerow([i, par, score, score - par])
        writer.writerow(["TOTAL", sum(pars), sum(scores), sum(scores) - sum(pars)])
    print(f"  CSV saved: {csv_path}")
    return csv_path


def _prompt_18_values(label: str, prefill: list = None) -> list:
    """Prompt the user to enter 18 integer values, one per hole."""
    print(f"\nEnter {label} for each hole (1-18), separated by spaces.")
    if prefill:
        print(f"  Pre-filled: {' '.join(str(v) for v in prefill)}")
        answer = input("  Press Enter to accept, or type new values: ").strip()
        if not answer:
            return list(prefill)
    else:
        answer = input("  Values: ").strip()

    parts = answer.split()
    values = []
    for p in parts:
        try:
            values.append(int(p))
        except ValueError:
            print(f"  Skipping non-integer: {p}")

    if len(values) != 18:
        print(f"  Got {len(values)} values, need exactly 18. Please re-enter.")
        return _prompt_18_values(label)

    return values


def _review_course(course: dict) -> dict:
    """Let the user confirm or correct course data."""
    print(f"\n--- COURSE ---")
    print(f"  Detected: {course['name']}")
    print(f"  Par: {course['par']}  (front {course['front']} / back {course['back']})")
    print(f"  Hole pars: {' '.join(str(p) for p in course['holePars'])}")

    name = input(f"\nCourse name [{course['name']}]: ").strip() or course['name']
    while not name:
        name = input("Course name: ").strip()

    confirm = input("Accept hole pars above? (y/n): ").strip().lower()
    if confirm != "y":
        hole_pars = _prompt_18_values("par values", prefill=course["holePars"])
    else:
        hole_pars = course["holePars"]

    return {"name": name, "holePars": hole_pars}


def _review_player(player: dict) -> dict:
    """Let the user confirm or correct a player's scores."""
    print(f"\n--- PLAYER: {player['name']} ---")
    print(f"  Detected total: {player['total']}")
    print(f"  Scores: {' '.join(str(s) for s in player['scores'])}")

    name = input(f"Player name [{player['name']}]: ").strip() or player["name"]
    confirm = input("Accept scores above? (y/n): ").strip().lower()
    if confirm != "y":
        scores = _prompt_18_values("scores", prefill=player["scores"])
    else:
        scores = player["scores"]

    return {"name": name, "scores": scores}


def scan_and_store(image_path: str) -> None:
    """
    Full pipeline:
      1. Send image to Gemini 2.5 Flash
      2. Parse structured JSON (course + players)
      3. User reviews and confirms/corrects each section
      4. Save one Scorecard per player to the database
      5. Write per-player CSV
      6. Print analytics per player
    """
    if not os.path.exists(image_path):
        print(f"Error: image not found: {image_path}")
        sys.exit(1)

    # Step 1: Call Gemini
    data = _call_gemini(image_path)

    print("\n--- GEMINI EXTRACTED DATA ---")
    print(json.dumps(data, indent=2))

    # Step 2: Review course
    course = _review_course(data["course"])

    # Step 3: Review and save each player
    db = SessionLocal()
    try:
        for raw_player in data["players"]:
            player = _review_player(raw_player)

            print(f"\nSaving scorecard for '{player['name']}' at '{course['name']}'...")
            scorecard = save_scorecard_to_db(
                db=db,
                username=player["name"],
                course_name=course["name"],
                scores=player["scores"],
                course_pars=course["holePars"],
                image_path=image_path,
                raw_ocr_data=data,
            )
            print(f"  Saved! Scorecard ID: {scorecard.id}")

            _write_scan_csv(image_path, player["name"], player["scores"], course["holePars"], course["name"])
            print_user_breakdown(player["name"])
    finally:
        db.close()
