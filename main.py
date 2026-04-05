#!/usr/bin/env python
"""Entry point for golf scorecard tracking system."""
import sys
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent))

from src.api.server import app
from src.analysis.stats import print_user_breakdown, print_all_users


def main():
    """Main entry point."""
    if len(sys.argv) < 2:
        print("Golf Scorecard Tracking System")
        print("\nUsage:")
        print("  python main.py api               # Start REST API server")
        print("  python main.py stats <username>  # Show user statistics")
        return
    
    command = sys.argv[1].lower()
    
    if command == "api":
        print("Starting API server on http://localhost:8000")
        import uvicorn
        uvicorn.run(app, host="0.0.0.0", port=8000)


if __name__ == "__main__":
    main()
