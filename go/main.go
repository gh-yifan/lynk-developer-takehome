// Spacecraft Command Queue API
//
// Implement a priority-based command queue for spacecraft operations.
// See ../README.md for full requirements.

package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// TODO: Define your data structures here
// Hint: You'll need structs for Command, Priority, Status, etc.
// Example:
type Priority string

const (
	PriorityHigh    Priority = "HIGH"
	PriorityMedium  Priority = "MEDIUM"
	PriorityLow     Priority = "LOW"
	StatusPending   Status   = "PENDING"
	StatusExecuting Status   = "EXECUTING"
	StatusCompleted Status   = "COMPLETED"
	StatusFailed    Status   = "FAILED"
)

type Status string

type CommandReqPost struct {
	Command  string   `json:"command" validate:"required,min=1,max=198"`
	Priority Priority `json:"priority" validate:"required,oneof=HIGH MEDIUM LOW"`
}

type CommandReqPatch struct {
	Status Status `json:"status" validate:"required,oneof=PENDING EXECUTING COMPLETED FAILED"`
}

// capital or lowercase?
type Command struct {
	Id        uuid.UUID `json:"id"`
	Command   string    `json:"command"`
	Priority  Priority  `json:"priority"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CommandStorage struct {
	mu     sync.RWMutex
	queue  map[Priority]*list.List     // 3 keys with growing list, points to the command in the direct map
	direct map[uuid.UUID]*list.Element // command id -> pointer to command
}

var valid = map[Status][]Status{
	StatusPending:   []Status{StatusExecuting},
	StatusExecuting: []Status{StatusCompleted, StatusFailed},
}

var validate = validator.New()

// GetCommands handles GET /commands
// Get all commands in priority order (HIGH -> MEDIUM -> LOW).
// Within the same priority, return in FIFO order.
//
// TODO: Implement this handler
func (c *CommandStorage) GetCommands(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	commands := make([]*Command, 0,
		c.queue[PriorityHigh].Len()+c.queue[PriorityMedium].Len()+c.queue[PriorityLow].Len())
	appendCommands := func(queue *list.List) {
		for element := queue.Front(); element != nil; element = element.Next() {
			commands = append(commands, element.Value.(*Command))
		}
	}
	appendCommands(c.queue[PriorityHigh])
	appendCommands(c.queue[PriorityMedium])
	appendCommands(c.queue[PriorityLow])

	response := map[string]interface{}{
		"commands": commands,
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
func (c *CommandStorage) CreateCommand(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandReqPost
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		writeJSONError(w, fmt.Sprintf("Invalid request: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// validate priority is one of enum
	// validate command is not empty and within char limits

	cmdId := uuid.New()

	cmd := &Command{
		Id:        cmdId,
		Priority:  req.Priority,
		Command:   req.Command,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	element := c.queue[req.Priority].PushBack(cmd)
	c.direct[cmdId] = element

	// TODO: Implement command creation
	response := map[string]interface{}{
		"command": cmd,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (c *CommandStorage) PatchCommand(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if r.Method != http.MethodPatch {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandReqPatch
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	elem, ok := c.direct[id]
	if err != nil || !ok {
		writeJSONError(w, "Command not found", http.StatusNotFound)
		return
	}

	cmd := elem.Value.(*Command)
	// validate state transition
	var found bool
	if acceptable, ok := valid[cmd.Status]; ok {
		for _, s := range acceptable {
			if req.Status == s {
				// is in the accepted range
				found = true
			}
		}
	}

	if !found {
		writeJSONError(w, fmt.Sprintf("Invalid state transition: %s -> %s", cmd.Status, req.Status), http.StatusBadRequest)
		return
	}

	cmd.Status = req.Status
	cmd.UpdatedAt = time.Now()
	response := map[string]interface{}{
		"command": cmd,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *CommandStorage) DeleteCommand(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if r.Method != http.MethodDelete {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	elem, ok := c.direct[id]
	if err != nil || !ok {
		writeJSONError(w, "Command not found", http.StatusNotFound)
		return
	}

	cmd := elem.Value.(*Command)
	// remove the command from the priority queue while maintaining FIFO order
	c.queue[cmd.Priority].Remove(elem) // list isn't a container itself, just the pointers for all nodes that belong to it

	delete(c.direct, id)

	response := map[string]interface{}{
		"message": "Command deleted",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TODO: Add more handlers as needed:
// - UpdateCommandStatus - PATCH /commands/{id}/status
// - DeleteCommand - DELETE /commands/{id}
// - GetCommand - GET /commands/{id} (optional)

func main() {
	storage := &CommandStorage{
		queue:  make(map[Priority]*list.List),
		direct: make(map[uuid.UUID]*list.Element),
	}
	storage.queue[PriorityHigh] = list.New()
	storage.queue[PriorityMedium] = list.New()
	storage.queue[PriorityLow] = list.New()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /commands", storage.GetCommands)
	mux.HandleFunc("POST /commands", storage.CreateCommand)
	mux.HandleFunc("PATCH /commands/{id}/status", storage.PatchCommand)
	mux.HandleFunc("DELETE /commands/{id}", storage.DeleteCommand)

	fmt.Println("Command Queue API running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
