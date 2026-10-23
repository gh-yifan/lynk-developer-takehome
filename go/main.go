// Spacecraft Command Queue API
//
// Implement a priority-based command queue for spacecraft operations.
// See ../README.md for full requirements.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// TODO: Define your data structures here
// Hint: You'll need structs for Command, Priority, Status, etc.
// Example:
// type Priority string
// const (
//     PriorityHigh   Priority = "HIGH"
//     PriorityMedium Priority = "MEDIUM"
//     PriorityLow    Priority = "LOW"
// )

// GetCommands handles GET /commands
// Get all commands in priority order (HIGH -> MEDIUM -> LOW).
// Within the same priority, return in FIFO order.
//
// TODO: Implement this handler
func GetCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement - return all commands in priority order
	response := map[string]interface{}{
		"commands": []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateCommand handles POST /commands
// Create a new command in the queue.
//
// TODO: Implement this handler
// - Validate command text (non-empty, <= 198 chars)
// - Validate priority (HIGH, MEDIUM, LOW)
// - Assign ID, status=PENDING, timestamps
// - Add to queue
func CreateCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement command creation
	response := map[string]interface{}{
		"command": map[string]interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// TODO: Add more handlers as needed:
// - UpdateCommandStatus - PATCH /commands/{id}/status
// - DeleteCommand - DELETE /commands/{id}
// - GetCommand - GET /commands/{id} (optional)

func main() {
	http.HandleFunc("/commands", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetCommands(w, r)
		case http.MethodPost:
			CreateCommand(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Command Queue API running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
