"""
Optional Storage Helper

This module provides simple file-based persistence for the command queue.
You can use this if you want your queue to survive server restarts.

Usage is completely optional - in-memory storage is acceptable.

Example usage:
    from storage import save_queue, load_queue

    # Load queue on startup
    commands = load_queue()

    # Save queue after modifications
    save_queue(commands)
"""

import json
from typing import List, Any

STORAGE_FILE = "queue_data.json"


def save_queue(commands: List[Any]) -> None:
    """
    Save the command queue to a JSON file.

    Args:
        commands: List of command objects (as dicts)
    """
    with open(STORAGE_FILE, 'w') as f:
        json.dump(commands, f, indent=2)


def load_queue() -> List[Any]:
    """
    Load the command queue from a JSON file.

    Returns:
        List of command objects (as dicts), or empty list if file doesn't exist
    """
    try:
        with open(STORAGE_FILE, 'r') as f:
            return json.load(f)
    except FileNotFoundError:
        return []
