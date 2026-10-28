package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestCommandStorage() *CommandStorage {
	return &CommandStorage{
		queue: map[Priority]*list.List{
			PriorityHigh:   list.New(),
			PriorityMedium: list.New(),
			PriorityLow:    list.New(),
		},
		pending: map[Priority]*list.List{
			PriorityHigh:   list.New(),
			PriorityMedium: list.New(),
			PriorityLow:    list.New(),
		},
		direct:        make(map[uuid.UUID]*list.Element),
		pendingDirect: make(map[uuid.UUID]*list.Element),
	}
}

func addTestCommand(storage *CommandStorage, priority Priority, status Status, text string) *Command {
	command := &Command{
		Id:        uuid.New(),
		Command:   text,
		Priority:  priority,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	storage.direct[command.Id] = storage.queue[priority].PushBack(command)
	if status == StatusPending {
		storage.pendingDirect[command.Id] = storage.pending[priority].PushBack(command)
	}
	return command
}

func createCommandForTest(t *testing.T, storage *CommandStorage, command string, priority Priority) Command {
	t.Helper()
	body := fmt.Sprintf(`{"command":%q,"priority":%q}`, command, priority)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/commands", strings.NewReader(body))
	storage.CreateCommand(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create command returned status %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Command Command `json:"command"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	return response.Command
}

func dispatchForTest(t *testing.T, storage *CommandStorage) (*Command, int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/commands/dispatch", nil)
	storage.DispatchNext(recorder, request)
	if recorder.Code != http.StatusOK {
		return nil, recorder.Code
	}
	var response struct {
		Command Command `json:"command"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode dispatch response: %v", err)
	}
	return &response.Command, recorder.Code
}

func TestDispatchNextEmptyQueueReturnsNotFound(t *testing.T) {
	storage := newTestCommandStorage()
	_, status := dispatchForTest(t, storage)
	if status != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, status)
	}
}

func TestDispatchNextDispatchesLowPriorityCommand(t *testing.T) {
	storage := newTestCommandStorage()
	want := createCommandForTest(t, storage, "low task", PriorityLow)
	got, status := dispatchForTest(t, storage)
	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}
	if got.Id != want.Id || got.Status != StatusExecuting {
		t.Fatalf("expected LOW command %s to be executing, got ID %s status %s", want.Id, got.Id, got.Status)
	}
}

func TestDispatchNextUsesPriorityThenReturnsNotFound(t *testing.T) {
	storage := newTestCommandStorage()
	low := createCommandForTest(t, storage, "low task", PriorityLow)
	high := createCommandForTest(t, storage, "high task", PriorityHigh)

	for _, want := range []Command{high, low} {
		got, status := dispatchForTest(t, storage)
		if status != http.StatusOK {
			t.Fatalf("expected dispatch status %d, got %d", http.StatusOK, status)
		}
		if got.Id != want.Id {
			t.Fatalf("expected %s command %s, got %s", want.Priority, want.Id, got.Id)
		}
	}
	if _, status := dispatchForTest(t, storage); status != http.StatusNotFound {
		t.Fatalf("expected exhausted queue status %d, got %d", http.StatusNotFound, status)
	}
}

func TestDispatchNextPicksNewHigherPriorityCommand(t *testing.T) {
	storage := newTestCommandStorage()
	medium := createCommandForTest(t, storage, "medium task", PriorityMedium)
	low := createCommandForTest(t, storage, "low task", PriorityLow)

	if got, status := dispatchForTest(t, storage); status != http.StatusOK || got.Id != medium.Id {
		t.Fatalf("expected initial MEDIUM command %s, got status %d command %v", medium.Id, status, got)
	}
	high := createCommandForTest(t, storage, "new high task", PriorityHigh)
	if got, status := dispatchForTest(t, storage); status != http.StatusOK || got.Id != high.Id {
		t.Fatalf("expected newly created HIGH command %s, got status %d command %v", high.Id, status, got)
	}
	if got, status := dispatchForTest(t, storage); status != http.StatusOK || got.Id != low.Id {
		t.Fatalf("expected LOW command %s next, got status %d command %v", low.Id, status, got)
	}
}

func TestDispatchNextLowerPriorityDoesNotOvertakeHigherPriority(t *testing.T) {
	storage := newTestCommandStorage()
	high := createCommandForTest(t, storage, "high task", PriorityHigh)
	medium := createCommandForTest(t, storage, "medium task", PriorityMedium)

	if got, status := dispatchForTest(t, storage); status != http.StatusOK || got.Id != high.Id {
		t.Fatalf("expected HIGH command %s first, got status %d command %v", high.Id, status, got)
	}
	low := createCommandForTest(t, storage, "new low task", PriorityLow)
	if got, status := dispatchForTest(t, storage); status != http.StatusOK || got.Id != medium.Id {
		t.Fatalf("expected MEDIUM command %s before LOW command %s, got status %d command %v", medium.Id, low.Id, status, got)
	}
	if got, status := dispatchForTest(t, storage); status != http.StatusOK || got.Id != low.Id {
		t.Fatalf("expected LOW command %s after MEDIUM, got status %d command %v", low.Id, status, got)
	}
}

func TestDispatchNextSelectsHighestPriorityPendingInFIFOOrder(t *testing.T) {
	storage := newTestCommandStorage()
	addTestCommand(storage, PriorityHigh, StatusExecuting, "already executing")
	firstHigh := addTestCommand(storage, PriorityHigh, StatusPending, "first high")
	addTestCommand(storage, PriorityHigh, StatusPending, "second high")
	addTestCommand(storage, PriorityMedium, StatusPending, "medium")
	addTestCommand(storage, PriorityLow, StatusPending, "low")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/commands/dispatch", nil)
	storage.DispatchNext(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Command Command `json:"command"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Command.Id != firstHigh.Id {
		t.Fatalf("expected command %s, got %s", firstHigh.Id, response.Command.Id)
	}
	if firstHigh.Status != StatusExecuting {
		t.Fatalf("expected selected command status %s, got %s", StatusExecuting, firstHigh.Status)
	}
}

func TestDispatchNextReturnsNotFoundWhenNoPendingCommands(t *testing.T) {
	storage := newTestCommandStorage()
	addTestCommand(storage, PriorityHigh, StatusExecuting, "already executing")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/commands/dispatch", nil)
	storage.DispatchNext(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestPatchCommandInvalidTransitionReturnsConflict(t *testing.T) {
	storage := newTestCommandStorage()
	command := addTestCommand(storage, PriorityHigh, StatusCompleted, "completed command")
	request := httptest.NewRequest(http.MethodPatch, "/commands/"+command.Id.String()+"/status", strings.NewReader(`{"status":"EXECUTING"}`))
	recorder := httptest.NewRecorder()
	newRouter(storage).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
}

func TestOpenAPISpecEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	recorder := httptest.NewRecorder()
	newRouter(newTestCommandStorage()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/yaml" {
		t.Fatalf("expected YAML content type, got %q", recorder.Header().Get("Content-Type"))
	}
	if !strings.Contains(recorder.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("expected OpenAPI YAML body, got %q", recorder.Body.String())
	}
}

func TestCreateCommandValidationAndTrim(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "blank command",
			body:       `{"command":"   ","priority":"LOW"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "command must not be blank",
		},
		{
			name:       "command too long",
			body:       `{"command":"` + strings.Repeat("x", 199) + `","priority":"LOW"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "command must be at most 198 characters",
		},
		{
			name:       "invalid priority",
			body:       `{"command":"execute","priority":"URGENT"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "priority must be HIGH, MEDIUM, or LOW",
		},
		{
			name:       "trims surrounding whitespace",
			body:       `{"command":"  execute  ","priority":"LOW"}`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := newTestCommandStorage()
			request := httptest.NewRequest(http.MethodPost, "/commands", strings.NewReader(test.body))
			response := httptest.NewRecorder()

			storage.CreateCommand(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantError != "" {
				var body map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if body["error"] != test.wantError {
					t.Fatalf("error = %q, want %q", body["error"], test.wantError)
				}
				return
			}

			command := storage.queue[PriorityLow].Front().Value.(*Command)
			if command.Command != "execute" {
				t.Errorf("stored command = %q, want %q", command.Command, "execute")
			}
		})
	}
}

func TestGetCommandsWhileCreateBodyIsStalled(t *testing.T) {
	storage := newTestCommandStorage()
	bodyReader, bodyWriter := io.Pipe()
	defer bodyReader.Close()
	defer bodyWriter.Close()

	createResponse := httptest.NewRecorder()
	createDone := make(chan struct{})
	go func() {
		request := httptest.NewRequest(http.MethodPost, "/commands", bodyReader)
		storage.CreateCommand(createResponse, request)
		close(createDone)
	}()

	bodyWritten := make(chan error, 1)
	go func() {
		_, err := io.WriteString(bodyWriter, `{"command":`)
		bodyWritten <- err
	}()
	select {
	case err := <-bodyWritten:
		if err != nil {
			t.Fatalf("write partial POST body: %v", err)
		}
	case <-time.After(time.Second):
		_ = bodyWriter.Close()
		t.Fatal("POST handler did not read the partial body")
	}

	getResponse := httptest.NewRecorder()
	getDone := make(chan struct{})
	go func() {
		request := httptest.NewRequest(http.MethodGet, "/commands", nil)
		storage.GetCommands(getResponse, request)
		close(getDone)
	}()

	select {
	case <-getDone:
	case <-time.After(time.Second):
		_ = bodyWriter.Close()
		<-createDone
		t.Fatal("GET /commands blocked while POST body was incomplete")
	}

	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	_ = bodyWriter.Close()
	select {
	case <-createDone:
	case <-time.After(time.Second):
		t.Fatal("POST handler did not finish after its body was closed")
	}
	if createResponse.Code != http.StatusBadRequest {
		t.Fatalf("POST status = %d, want %d", createResponse.Code, http.StatusBadRequest)
	}
}
