//! Spacecraft Command Queue API
//!
//! Implement a priority-based command queue for spacecraft operations.
//! See ../README.md for full requirements.

use axum::{
    extract::Json,
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Router,
};
use serde::{Deserialize, Serialize};

// TODO: Define your data structures here
// Hint: You'll need structs for Command, Priority, Status, etc.

#[derive(Serialize, Deserialize)]
struct CommandsResponse {
    commands: Vec<serde_json::Value>,
}

#[derive(Serialize, Deserialize)]
struct CommandResponse {
    command: serde_json::Value,
}

/// Get all commands in priority order (HIGH -> MEDIUM -> LOW).
/// Within the same priority, return in FIFO order.
///
/// TODO: Implement this handler
async fn get_commands() -> impl IntoResponse {
    // TODO: Implement - return all commands in priority order
    Json(CommandsResponse {
        commands: vec![],
    })
}

/// Create a new command in the queue.
///
/// TODO: Implement this handler
/// - Validate command text (non-empty, <= 198 chars)
/// - Validate priority (HIGH, MEDIUM, LOW)
/// - Assign ID, status=PENDING, timestamps
/// - Add to queue
async fn create_command(Json(payload): Json<serde_json::Value>) -> impl IntoResponse {
    // TODO: Implement command creation
    (
        StatusCode::CREATED,
        Json(CommandResponse {
            command: serde_json::json!({}),
        }),
    )
}

// TODO: Add more handlers as needed:
// - PATCH /commands/:id/status - Update command status
// - DELETE /commands/:id - Delete a command
// - GET /commands/:id - Get a specific command (optional)

#[tokio::main]
async fn main() {
    let app = Router::new()
        .route("/commands", get(get_commands))
        .route("/commands", post(create_command));

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080")
        .await
        .unwrap();

    println!("Command Queue API running on http://localhost:8080");

    axum::serve(listener, app).await.unwrap();
}
