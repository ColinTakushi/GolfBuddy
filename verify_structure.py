#!/usr/bin/env python
"""Verify the reorganized project structure works correctly."""

import sys
import os

os.chdir(os.path.dirname(os.path.abspath(__file__)))

print("🔍 VERIFYING PROJECT REORGANIZATION\n")
print("=" * 60)

tests = [
    ("Core Database Module", lambda: __import__('src.core.db', fromlist=['SessionLocal'])),
    ("ORM Models", lambda: __import__('src.core.models', fromlist=['User', 'Scorecard'])),
    ("Database Migration", lambda: __import__('src.database.migration', fromlist=['migrate_from_csv'])),
    ("REST API Server", lambda: __import__('src.api.server', fromlist=['app'])),
    ("OCR Processor", lambda: __import__('src.ocr.processor', fromlist=['process_scorecard_image'])),
    ("OCR Utils", lambda: __import__('src.ocr.utils', fromlist=['save_scorecard_to_db'])),
    ("Analytics Tool", lambda: __import__('tools.analytics', fromlist=['get_user_breakdown'])),
]

passed = 0
failed = 0

for name, test_func in tests:
    try:
        test_func()
        print(f"✅ {name:<30} OK")
        passed += 1
    except Exception as e:
        print(f"❌ {name:<30} ERROR: {str(e)[:40]}")
        failed += 1

print("=" * 60)
print(f"\nResults: {passed} passed, {failed} failed")

if failed == 0:
    print("\n🎉 All modules imported successfully!")
    print("\nAvailable commands:")
    print("  python analyze.py <username>     - Analyze user scores")
    print("  python server.py [port]          - Start API server")
    print("  python run_migration.py          - Migrate CSV data")
    sys.exit(0)
else:
    print(f"\n⚠️  {failed} module(s) failed to import")
    sys.exit(1)
