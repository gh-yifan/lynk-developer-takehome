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
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
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
	Command  string   `json:"command" validate:"notblank,max=198"`
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
	mu            sync.RWMutex
	queue         map[Priority]*list.List     // All commands, retained for ordered listing.
	pending       map[Priority]*list.List     // Dispatch-ready commands only.
	direct        map[uuid.UUID]*list.Element // Command ID -> element in queue.
	pendingDirect map[uuid.UUID]*list.Element // Pending command ID -> element in pending queue.
}

var valid = map[Status][]Status{
	StatusPending:   []Status{StatusExecuting},
	StatusExecuting: []Status{StatusCompleted, StatusFailed},
}

func newValidator() *validator.Validate {
	validate := validator.New()
	if err := validate.RegisterValidation("notblank", validators.NotBlank); err != nil {
		panic(err)
	}
	return validate
}

var validate = newValidator()

func validationErrorMessage(err error) string {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok || len(validationErrors) == 0 {
		return "invalid request"
	}

	fieldError := validationErrors[0]
	switch fieldError.StructField() {
	case "Command":
		switch fieldError.Tag() {
		case "notblank":
			return "command must not be blank"
		case "max":
			return "command must be at most 198 characters"
		}
	case "Priority":
		if fieldError.Tag() == "required" {
			return "priority is required"
		}
		return "priority must be HIGH, MEDIUM, or LOW"
	case "Status":
		if fieldError.Tag() == "required" {
			return "status is required"
		}
		return "status must be PENDING, EXECUTING, COMPLETED, or FAILED"
	}
	return "invalid request"
}

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

// DispatchNext handles POST /commands/dispatch and starts the next pending command.
func (c *CommandStorage) DispatchNext(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	for _, priority := range []Priority{PriorityHigh, PriorityMedium, PriorityLow} {
		element := c.pending[priority].Front()
		if element == nil {
			continue
		}

		cmd := element.Value.(*Command)
		c.pending[priority].Remove(element)
		delete(c.pendingDirect, cmd.Id)
		cmd.Status = StatusExecuting
		cmd.UpdatedAt = time.Now()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"command": cmd})
		return
	}

	writeJSONError(w, "No pending commands", http.StatusNotFound)
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
	req.Command = strings.TrimSpace(req.Command)

	if err := validate.Struct(req); err != nil {
		writeJSONError(w, validationErrorMessage(err), http.StatusBadRequest)
		return
	}

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
	pendingElement := c.pending[req.Priority].PushBack(cmd)
	c.pendingDirect[cmdId] = pendingElement

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
	if err := validate.Struct(req); err != nil {
		writeJSONError(w, validationErrorMessage(err), http.StatusBadRequest)
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
		writeJSONError(w, "invalid state transition", http.StatusBadRequest)
		return
	}
	if cmd.Status == StatusPending && req.Status == StatusExecuting {
		c.pending[cmd.Priority].Remove(c.pendingDirect[id])
		delete(c.pendingDirect, id)
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
	if cmd.Status == StatusPending {
		c.pending[cmd.Priority].Remove(c.pendingDirect[id])
		delete(c.pendingDirect, id)
	}

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
		queue:         make(map[Priority]*list.List),
		pending:       make(map[Priority]*list.List),
		direct:        make(map[uuid.UUID]*list.Element),
		pendingDirect: make(map[uuid.UUID]*list.Element),
	}
	storage.queue[PriorityHigh] = list.New()
	storage.queue[PriorityMedium] = list.New()
	storage.queue[PriorityLow] = list.New()
	storage.pending[PriorityHigh] = list.New()
	storage.pending[PriorityMedium] = list.New()
	storage.pending[PriorityLow] = list.New()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /commands", storage.GetCommands)
	mux.HandleFunc("POST /commands/dispatch", storage.DispatchNext)
	mux.HandleFunc("POST /commands", storage.CreateCommand)
	mux.HandleFunc("PATCH /commands/{id}/status", storage.PatchCommand)
	mux.HandleFunc("DELETE /commands/{id}", storage.DeleteCommand)

	fmt.Println("Command Queue API running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
