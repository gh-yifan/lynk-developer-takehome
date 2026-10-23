//! Optional Storage Helper
//!
//! This module provides simple file-based persistence for the command queue.
//! You can use this if you want your queue to survive server restarts.
//!
//! Usage is completely optional - in-memory storage is acceptable.
//!
//! Example usage:
//!   use crate::storage::{save_queue, load_queue};
//!
//!   // Load queue on startup
//!   let commands = load_queue()?;
//!
//!   // Save queue after modifications
//!   save_queue(&commands)?;

use serde::{Deserialize, Serialize};
use std::fs;
use std::io;

const STORAGE_FILE: &str = "queue_data.json";

/// Save the command queue to a JSON file.
pub fn save_queue<T: Serialize>(commands: &[T]) -> io::Result<()> {
    let json = serde_json::to_string_pretty(commands)?;
    fs::write(STORAGE_FILE, json)?;
    Ok(())
}

/// Load the command queue from a JSON file.
///
/// Returns an empty vector if the file doesn't exist.
pub fn load_queue<T: for<'de> Deserialize<'de>>() -> io::Result<Vec<T>> {
    match fs::read_to_string(STORAGE_FILE) {
        Ok(contents) => {
            let commands = serde_json::from_str(&contents)?;
            Ok(commands)
        }
        Err(e) if e.kind() == io::ErrorKind::NotFound => Ok(vec![]),
        Err(e) => Err(e),
    }
}
