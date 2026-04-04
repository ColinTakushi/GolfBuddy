#!/usr/bin/env python
"""Entry point for golf scorecard tracking system."""
import sys
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent))

from src.api.main import app
from src.analysis.stats import print_user_breakdown, print_all_users
from src.database.migrate import migrate_from_csv


def main():
    """Main entry point."""
    if len(sys.argv) < 2:
        print("Golf Scorecard Tracking System")
        print("\nUsage:")
        print("  python main.py api               # Start REST API server")
        print("  python main.py stats <username>  # Show user statistics")
        print("  python main.py stats             # List all users")
        print("  python main.py migrate           # Migrate CSV data to database")
        return
    
    command = sys.argv[1].lower()
    
    if command == "api":
        print("Starting API server on http://localhost:8000")
        import uvicorn
        uvicorn.run(app, host="0.0.0.0", port=8000)
    
    elif command == "stats":
        if len(sys.argv) > 2:
            username = sys.argv[2]
            print_user_breakdown(username)
        else:
            print("\nGolf Scorecard Analysis Tool")
            print("='*50")
            print("\nAvailable users:")
            from src.database.db import SessionLocal
            from src.database.models import User
            db = SessionLocal()
            try:
                users = db.query(User).all()
                for user in users:
                    rounds = len(user.scorecards)
                    print(f"  - {user.username} ({rounds} round{'s' if rounds != 1 else ''})")
                print("\nUsage: python main.py stats <username>")
            finally:
                db.close()
    
    elif command == "migrate":
        print("Migrating CSV data to database...")
        migrate_from_csv()
    
    else:
        print(f"Unknown command: {command}")
        print("Try: api, stats, or migrate")


if __name__ == "__main__":
    main()
