#!/usr/bin/env python3
"""Scan a scorecard image or load from JSON: review → save → analyze.

Usage:
    python3 scan.py image <image_path>
    python3 scan.py json <json_path>
"""
import sys
from src.ocr.pipeline import scan_and_store, store_json


def main():
    if len(sys.argv) < 3:
        print("Usage:")
        print("  python3 scan.py image <image_path>")
        print("  python3 scan.py json <json_path>")
        sys.exit(1)

    command = sys.argv[1].lower()
    path = sys.argv[2]

    if command == "image":
        print("Starting scan of image, sending to Gemini...")
        scan_and_store(path)
    elif command == "json":
        print("Using pre-scanned values from JSON...")
        store_json(path)
    else:
        print(f"Unknown command: {command}")
        print("Use 'image' or 'json'")
        sys.exit(1)


if __name__ == "__main__":
    main()
