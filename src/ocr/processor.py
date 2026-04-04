"""OCR processing module for golf scorecard images."""
import cv2
import easyocr
import numpy as np
from pathlib import Path
from typing import List, Tuple, Dict

# Initialize OCR reader (GPU enabled if available)
_reader = None

def get_reader():
    """Get or initialize the EasyOCR reader."""
    global _reader
    if _reader is None:
        _reader = easyocr.Reader(['en'], gpu=True)
    return _reader


# Image preprocessing functions

def get_grayscale(image):
    """Convert image to grayscale."""
    return cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)


def remove_noise(image):
    """Remove noise using median blur."""
    return cv2.medianBlur(image, 5)


def thresholding(image):
    """Apply automatic thresholding."""
    return cv2.threshold(image, 0, 255, cv2.THRESH_BINARY + cv2.THRESH_OTSU)[1]


def dilate(image):
    """Apply dilation morphological operation."""
    kernel = np.ones((5, 5), np.uint8)
    return cv2.dilate(image, kernel, iterations=1)


def erode(image):
    """Apply erosion morphological operation."""
    kernel = np.ones((5, 5), np.uint8)
    return cv2.erode(image, kernel, iterations=1)


def opening(image):
    """Apply opening morphological operation (erosion followed by dilation)."""
    kernel = np.ones((5, 5), np.uint8)
    return cv2.morphologyEx(image, cv2.MORPH_OPEN, kernel)


def canny(image):
    """Apply Canny edge detection."""
    return cv2.Canny(image, 100, 200)


def deskew(image):
    """Correct image skew/rotation."""
    coords = np.column_stack(np.where(image > 0))
    angle = cv2.minAreaRect(coords)[-1]
    if angle < -45:
        angle = -(90 + angle)
    else:
        angle = -angle
    (h, w) = image.shape[:2]
    center = (w // 2, h // 2)
    M = cv2.getRotationMatrix2D(center, angle, 1.0)
    rotated = cv2.warpAffine(
        image, M, (w, h),
        flags=cv2.INTER_CUBIC,
        borderMode=cv2.BORDER_REPLICATE
    )
    return rotated


def process_scorecard_image(
    image_path: str,
    apply_grayscale: bool = True,
    apply_noise_removal: bool = True,
    apply_opening: bool = True,
    apply_deskew: bool = False,
    output_dir: str = None
) -> Tuple[List[Tuple], float]:
    """
    Process a scorecard image and perform OCR.
    
    Args:
        image_path: Path to the scorecard image
        apply_grayscale: Convert to grayscale
        apply_noise_removal: Remove noise
        apply_opening: Apply morphological opening
        apply_deskew: Correct image skew
        output_dir: Optional directory to save processed image
    
    Returns:
        Tuple of (OCR results list, average confidence)
    """
    # Load image
    img = cv2.imread(image_path)
    if img is None:
        raise FileNotFoundError(f"Image not found: {image_path}")
    
    # Preprocess image
    processed = img.copy()
    
    if apply_grayscale:
        processed = get_grayscale(processed)
    
    if apply_noise_removal:
        processed = remove_noise(processed)
    
    if apply_deskew:
        processed = deskew(processed)
    
    if apply_opening:
        processed = opening(processed)
    
    # Perform OCR
    reader = get_reader()
    results = reader.readtext(processed, detail=1, paragraph=False)
    
    # Clean text (remove non-ASCII characters)
    cleaned_results = []
    for (bbox, text, confidence) in results:
        text = "".join([c if ord(c) < 128 else "" for c in text]).strip()
        cleaned_results.append((bbox, text, confidence))
    
    # Draw results on image
    output_image = processed.copy()
    if output_image.ndim == 2:  # Grayscale
        output_image = cv2.cvtColor(output_image, cv2.COLOR_GRAY2BGR)
    
    for (bbox, text, confidence) in cleaned_results:
        (tl, tr, br, bl) = bbox
        tl = (int(tl[0]), int(tl[1]))
        tr = (int(tr[0]), int(tr[1]))
        br = (int(br[0]), int(br[1]))
        bl = (int(bl[0]), int(bl[1]))
        
        cv2.rectangle(output_image, tl, br, (0, 255, 0), 2)
        cv2.putText(
            output_image, f"{text} ({confidence:.2f})",
            (tl[0], tl[1] - 10),
            cv2.FONT_HERSHEY_SIMPLEX, 0.8, (0, 255, 0), 2
        )
    
    # Save output image if requested
    if output_dir:
        output_path = Path(output_dir) / "ocr_output.png"
        output_path.parent.mkdir(parents=True, exist_ok=True)
        cv2.imwrite(str(output_path), output_image)
    
    # Calculate average confidence
    average_confidence = (
        sum([item[2] for item in cleaned_results]) / len(cleaned_results)
        if cleaned_results
        else 0.0
    )
    
    return cleaned_results, average_confidence


def extract_scorecard_data(
    ocr_results: List[Tuple],
    course_name: str = None
) -> Dict:
    """
    Extract structured scorecard data from OCR results.
    
    Args:
        ocr_results: List of (bbox, text, confidence) tuples from OCR
        course_name: Optional course name for the scorecard
    
    Returns:
        Dictionary with extracted scorecard data
    """
    data = {
        "course_name": course_name or "Unknown",
        "raw_ocr": [],
        "detected_scores": [],
        "detected_par": [],
        "confidence_stats": {}
    }
    
    confidences = []
    
    for (bbox, text, confidence) in ocr_results:
        data["raw_ocr"].append({
            "text": text,
            "confidence": confidence
        })
        confidences.append(confidence)
        
        # Try to parse as number (hole score or par)
        try:
            score = int(text)
            if 1 <= score <= 13:  # Reasonable golf score
                data["detected_scores"].append(score)
        except ValueError:
            pass
    
    if confidences:
        data["confidence_stats"]["average"] = sum(confidences) / len(confidences)
        data["confidence_stats"]["min"] = min(confidences)
        data["confidence_stats"]["max"] = max(confidences)
    
    return data
