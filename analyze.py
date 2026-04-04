#!/usr/bin/env python
"""Golf scorecard analysis CLI - wrapper for new project structure."""

import sys
from tools.analytics import print_user_breakdown, print_all_users

if __name__ == "__main__":
    if len(sys.argv) > 1:
        username = sys.argv[1]
        print_user_breakdown(username)
    else:
        print("Golf Scorecard Analysis Tool")
        print("Usage: python analyze.py <username>")
        print("\nAvailable users:")
        from src.core.db import SessionLocal
        from src.core.models import User
        db = SessionLocal()
        try:
            users = db.query(User).all()
            for user in users:
                rounds = len(user.scorecards)
                print(f"  - {user.username} ({rounds} round{'s' if rounds != 1 else ''})")
        finally:
            db.close()
