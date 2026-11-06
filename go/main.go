// Spacecraft Command Queue API
//
// Implement a priority-based command queue for spacecraft operations.
// See ../README.md for full requirements.

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
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
	mu      sync.RWMutex
	queue   map[Priority][]*Command // All commands, retained for ordered listing.
	pending map[Priority][]*Command // Dispatch-ready commands only.
	direct  map[uuid.UUID]*Command  // Command ID -> command for direct lookup.
}

var validTransitions = map[Status][]Status{
	StatusPending:   []Status{StatusExecuting},
	StatusExecuting: []Status{StatusCompleted, StatusFailed},
}

func (s Status) CanTransitionTo(next Status) bool {
	return slices.Contains(validTransitions[s], next) // nil slice for terminal states -> false
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
		len(c.queue[PriorityHigh])+len(c.queue[PriorityMedium])+len(c.queue[PriorityLow]))
	commands = append(commands, c.queue[PriorityHigh]...)
	commands = append(commands, c.queue[PriorityMedium]...)
	commands = append(commands, c.queue[PriorityLow]...)

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
		pending := c.pending[priority]
		if len(pending) == 0 {
			continue
		}

		cmd := pending[0]
		c.pending[priority] = slices.Delete(pending, 0, 1)
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

	// moved here to prevent Create from starving other operations while waiting for input
	c.mu.Lock()
	defer c.mu.Unlock()

	cmdId := uuid.New()

	cmd := &Command{
		Id:        cmdId,
		Priority:  req.Priority,
		Command:   req.Command,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	c.queue[req.Priority] = append(c.queue[req.Priority], cmd)
	c.direct[cmdId] = cmd
	c.pending[req.Priority] = append(c.pending[req.Priority], cmd)

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

	if r.Method != http.MethodPatch {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandReqPatch
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		message := "Invalid JSON body"
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			message = "only status is allowed"
		}
		writeJSONError(w, message, http.StatusBadRequest)
		return
	}
	if err := validate.Struct(req); err != nil {
		writeJSONError(w, validationErrorMessage(err), http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	cmd, ok := c.direct[id]
	if err != nil || !ok {
		writeJSONError(w, "Command not found", http.StatusNotFound)
		return
	}

	// validate state transition
	if !cmd.Status.CanTransitionTo(req.Status) {
		writeJSONError(w, "invalid state transition", http.StatusConflict)
		return
	}
	if cmd.Status == StatusPending && req.Status == StatusExecuting {
		c.pending[cmd.Priority] = removeCommand(c.pending[cmd.Priority], cmd)
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
	cmd, ok := c.direct[id]
	if err != nil || !ok {
		writeJSONError(w, "Command not found", http.StatusNotFound)
		return
	}

	c.queue[cmd.Priority] = removeCommand(c.queue[cmd.Priority], cmd)
	if cmd.Status == StatusPending {
		c.pending[cmd.Priority] = removeCommand(c.pending[cmd.Priority], cmd)
	}

	delete(c.direct, id)

	response := map[string]interface{}{
		"message": "Command deleted",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func removeCommand(queue []*Command, target *Command) []*Command {
	for index, command := range queue {
		if command == target {
			return slices.Delete(queue, index, index+1)
		}
	}
	return queue
}

//go:embed openapi.yaml
var openAPISpec []byte

func openAPISpecHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openAPISpec)
}

func openAPIDocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Spacecraft Command Queue API</title>
	<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
	<script>
		SwaggerUIBundle({ url: "/openapi.yaml", dom_id: "#swagger-ui" });
	</script>
</body>
</html>`)
}

//go:embed demo.html
var queueDemoHTML []byte

func queueDemoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(queueDemoHTML)
}

func newRouter(storage *CommandStorage) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /demo", queueDemoHandler)
	mux.HandleFunc("GET /docs", openAPIDocsHandler)
	mux.HandleFunc("GET /openapi.yaml", openAPISpecHandler)
	mux.HandleFunc("GET /commands", storage.GetCommands)
	mux.HandleFunc("POST /commands/dispatch", storage.DispatchNext)
	mux.HandleFunc("POST /commands", storage.CreateCommand)
	mux.HandleFunc("PATCH /commands/{id}/status", storage.PatchCommand)
	mux.HandleFunc("DELETE /commands/{id}", storage.DeleteCommand)
	return mux
}

// TODO: Add more handlers as needed:
// - UpdateCommandStatus - PATCH /commands/{id}/status
// - DeleteCommand - DELETE /commands/{id}
// - GetCommand - GET /commands/{id} (optional)

func main() {
	storage := &CommandStorage{
		queue:   make(map[Priority][]*Command),
		pending: make(map[Priority][]*Command),
		direct:  make(map[uuid.UUID]*Command),
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Command Queue API running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, newRouter(storage)))
}
