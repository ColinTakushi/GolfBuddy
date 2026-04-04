#!/usr/bin/env python3
"""Scan a scorecard image: OCR → review → save → analyze.

Usage:
    python3 scan.py <image_path>

Example:
    python3 scan.py golf.jpg
"""
import sys
from src.ocr.pipeline import scan_and_store

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 scan.py <image_path>")
        sys.exit(1)
    scan_and_store(sys.argv[1])
