package main

import (
	"container/list"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestStorage() *CommandStorage {
	storage := &CommandStorage{
		queue:  make(map[Priority]*list.List),
		direct: make(map[uuid.UUID]*list.Element),
	}
	storage.queue[PriorityHigh] = list.New()
	storage.queue[PriorityMedium] = list.New()
	storage.queue[PriorityLow] = list.New()
	return storage
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
			storage := newTestStorage()
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
	storage := newTestStorage()
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
