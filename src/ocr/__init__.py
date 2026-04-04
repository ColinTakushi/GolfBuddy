"""OCR module for processing golf scorecard images."""
from src.ocr.utils import save_scorecard_to_db, create_image_storage_dir

__all__ = [
    "save_scorecard_to_db",
    "create_image_storage_dir",
]
