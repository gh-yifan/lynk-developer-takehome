"""
Spacecraft Command Queue API

Implement a priority-based command queue for spacecraft operations.
See ../README.md for full requirements.
"""

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import uvicorn

app = FastAPI(title="Command Queue API")


# TODO: Define your data structures here
# Hint: You'll need to represent commands with their properties


@app.get("/commands")
async def get_commands():
    """
    Get all commands in priority order (HIGH -> MEDIUM -> LOW).
    Within the same priority, return in FIFO order.

    TODO: Implement this endpoint
    Returns:
        {
            "commands": [array of command objects]
        }
    """
    # TODO: Implement - return all commands in priority order
    return {"commands": []}


@app.post("/commands", status_code=201)
async def create_command(request: dict):
    """
    Create a new command in the queue.

    TODO: Implement this endpoint
    - Validate command text (non-empty, <= 198 chars)
    - Validate priority (HIGH, MEDIUM, LOW)
    - Assign ID, status=PENDING, timestamps
    - Add to queue

    Request body:
        {
            "command": "command text here",
            "priority": "HIGH|MEDIUM|LOW"
        }

    Returns:
        {
            "command": {command object}
        }
    """
    # TODO: Implement command creation
    return {"command": {}}


# TODO: Add more endpoints as needed:
# - PATCH /commands/{id}/status - Update command status
# - DELETE /commands/{id} - Delete a command
# - GET /commands/{id} - Get a specific command (optional)


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
