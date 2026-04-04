#!/usr/bin/env python
"""Start the FastAPI server - wrapper for new project structure."""

import sys
import uvicorn
from src.api.server import app

if __name__ == "__main__":
    host = "0.0.0.0"
    port = 8000
    
    # Parse command line arguments
    if len(sys.argv) > 1:
        if sys.argv[1].isdigit():
            port = int(sys.argv[1])
    
    print(f"Starting Scorecard API on {host}:{port}")
    print(f"Documentation available at http://{host}:{port}/docs")
    
    uvicorn.run(app, host=host, port=port)
