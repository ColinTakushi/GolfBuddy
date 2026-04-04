"""Golf scorecard analysis and statistics tool."""
import numpy as np
from src.core.db import SessionLocal
from src.core.models import User, Scorecard
from typing import Optional


def get_user_breakdown(username: str, db: Optional[SessionLocal] = None) -> dict:
    """
    Get detailed scorecard breakdown for a user.
    
    Args:
        username: Username to get stats for
        db: Optional SQLAlchemy session (creates new one if not provided)
    
    Returns:
        Dictionary with user statistics
    """
    if db is None:
        db = SessionLocal()
        should_close = True
    else:
        should_close = False
    
    try:
        user = db.query(User).filter(User.username == username).first()
        if not user:
            return {"error": f"User '{username}' not found"}
        
        scorecards = user.scorecards
        if not scorecards:
            return {"error": f"No scorecards found for user '{username}'"}
        
        results = {
            "username": username,
            "scorecards": []
        }
        
        for sc in sorted(scorecards, key=lambda x: x.date):
            course = sc.course
            total_score = sc.get_total_score()
            total_par = sc.get_total_par()
            
            # Get hole breakdown
            breakdown = sc.get_hole_breakdown()
            
            scorecard_data = {
                "date": sc.date.strftime("%Y-%m-%d"),
                "course": course.name,
                "total_score": total_score,
                "total_par": total_par,
                "differential": total_score - total_par,
                "front_9": sc.get_front_9_score(),
                "front_9_par": course.get_front_9_par(),
                "back_9": sc.get_back_9_score(),
                "back_9_par": course.get_back_9_par(),
                "birdies": breakdown["birdies"],
                "pars": breakdown["pars"],
                "bogeys": breakdown["bogeys"],
                "doubles_plus": breakdown["doubles_plus"]
            }
            results["scorecards"].append(scorecard_data)
        
        # Calculate overall statistics
        all_scores = [sc.get_total_score() for sc in scorecards]
        all_differentials = [sc.get_score_differential() for sc in scorecards]
        
        results["summary"] = {
            "total_rounds": len(scorecards),
            "average_score": sum(all_scores) / len(all_scores),
            "best_score": min(all_scores),
            "worst_score": max(all_scores),
            "average_differential": sum(all_differentials) / len(all_differentials),
            "courses_played": len(set(sc.course_id for sc in scorecards))
        }
        
        return results
    
    finally:
        if should_close:
            db.close()


def print_user_breakdown(username: str):
    """Print formatted user breakdown to console."""
    stats = get_user_breakdown(username)
    
    if "error" in stats:
        print(f"Error: {stats['error']}")
        return
    
    print(f"\n{'='*70}")
    print(f"SCORECARD BREAKDOWN: {stats['username'].upper()}")
    print(f"{'='*70}")
    
    # Print summary
    summary = stats["summary"]
    print(f"\nSUMMARY:")
    print(f"  Total Rounds: {summary['total_rounds']}")
    print(f"  Courses Played: {summary['courses_played']}")
    print(f"  Average Score: {summary['average_score']:.1f}")
    print(f"  Best Score: {summary['best_score']}")
    print(f"  Worst Score: {summary['worst_score']}")
    print(f"  Average Differential: {summary['average_differential']:+.1f}")
    
    # Print individual scorecards
    print(f"\n{'='*70}")
    print("INDIVIDUAL ROUNDS:")
    print(f"{'='*70}\n")
    
    for i, sc_data in enumerate(stats["scorecards"], 1):
        print(f"Round {i}: {sc_data['date']} at {sc_data['course']}")
        print(f"  Total: {sc_data['total_score']} (vs. {sc_data['total_par']} par) {sc_data['differential']:+d}")
        print(f"  Front 9: {sc_data['front_9']} (vs. {sc_data['front_9_par']} par)")
        print(f"  Back 9: {sc_data['back_9']} (vs. {sc_data['back_9_par']} par)")
        print(f"  Breakdown: {sc_data['birdies']} Birdies, {sc_data['pars']} Pars, "
              f"{sc_data['bogeys']} Bogeys, {sc_data['doubles_plus']} Doubles+")
        print()


def print_round_summary(scorecard):
    """Print stats for a single scorecard round."""
    course = scorecard.course
    total_score = scorecard.get_total_score()
    total_par = scorecard.get_total_par()
    differential = total_score - total_par
    breakdown = scorecard.get_hole_breakdown()

    print(f"\n{'='*50}")
    print(f"ROUND SAVED: {scorecard.date.strftime('%Y-%m-%d')} at {course.name}")
    print(f"{'='*50}")
    print(f"  Total:   {total_score} (vs. {total_par} par)  {differential:+d}")
    print(f"  Front 9: {scorecard.get_front_9_score()} (vs. {course.get_front_9_par()} par)")
    print(f"  Back 9:  {scorecard.get_back_9_score()} (vs. {course.get_back_9_par()} par)")
    print(f"  Birdies: {breakdown['birdies']}  Pars: {breakdown['pars']}  "
          f"Bogeys: {breakdown['bogeys']}  Doubles+: {breakdown['doubles_plus']}")


def print_all_users():
    """Print breakdown for all users."""
    db = SessionLocal()
    try:
        users = db.query(User).all()
        for user in users:
            print_user_breakdown(user.username)
    finally:
        db.close()


if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1:
        username = sys.argv[1]
        print_user_breakdown(username)
    else:
        print("Golf Scorecard Analysis Tool")
        print("Usage: python analytics.py <username>")
        print("\nAvailable users:")
        db = SessionLocal()
        try:
            users = db.query(User).all()
            for user in users:
                rounds = len(user.scorecards)
                print(f"  - {user.username} ({rounds} round{'s' if rounds != 1 else ''})")
        finally:
            db.close()
