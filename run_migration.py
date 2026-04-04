#!/usr/bin/env python
"""Run database migration - wrapper for new project structure."""

from src.database.migration import migrate_from_csv

if __name__ == "__main__":
    migrate_from_csv()
