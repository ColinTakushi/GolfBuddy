#!/usr/bin/env python3
"""Scan a scorecard image or load from JSON: review → save → analyze.

Usage:
    python3 scan.py image <image_path>
    python3 scan.py image <image_path> --parse   # Gemini only, print JSON, no save
    python3 scan.py json <json_path>
    python3 scan.py save <json_path>             # save pre-reviewed JSON, no prompts
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))
from src.ocr.pipeline import scan_and_store, store_json, parse_image, save_from_data
import json


def main():
    if len(sys.argv) < 3:
        print("Usage:")
        print("  python3 scan.py image <image_path>")
        print("  python3 scan.py image <image_path> --parse")
        print("  python3 scan.py json <json_path>")
        print("  python3 scan.py save <json_path>")
        sys.exit(1)

    command = sys.argv[1].lower()
    path = sys.argv[2]
    flags = sys.argv[3:]

    if command == "image":
        if "--parse" in flags:
            data = parse_image(path)
            print(json.dumps(data))
        else:
            print("Starting scan of image, sending to Gemini...")
            scan_and_store(path)
    elif command == "json":
        print("Using pre-scanned values from JSON...")
        store_json(path)
    elif command == "save":
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
        save_from_data(data)
    else:
        print(f"Unknown command: {command}")
        print("Use 'image', 'json', or 'save'")
        sys.exit(1)


if __name__ == "__main__":
    main()
