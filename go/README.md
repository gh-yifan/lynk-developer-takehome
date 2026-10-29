# Go Implementation

This scaffold uses the standard library `net/http` package for the REST API.

## Setup

1. **Install Go 1.21 or higher** (if not already installed):
   - Check version: `go version`
   - Download from: https://go.dev/dl/

2. **Run the server:**
   ```bash
   go run main.go storage.go
   ```

   Or build and run:
   ```bash
   go build -o command-queue-api
   ./command-queue-api
   ```

   The server will start on `http://localhost:8080`

3. **Test it works:**
   ```bash
   curl http://localhost:8080/commands
   ```

## Interactive Docs and Queue Walkthrough

- Open `http://localhost:8080/demo` for the interactive queue UI, or `http://localhost:8080/docs` for Swagger UI.
- Run `bash ./demo.sh` in this directory to open the interactive queue UI. Use the controls to load sample commands, create commands, dispatch the next pending item, advance commands to completed or failed, delete them, and trigger representative API failures. Requests and responses appear in the activity panel. If an older API is occupying port 8080, the script starts the updated server on the first available port from 8081 through 8090.
- Run `make demo` from this directory to open the interactive UI; `make run` is an alias. Use `make deploy` to run the API in the foreground without opening a browser.

## Running Validation

From the parent directory:
```bash
cd ..
python3 validate.py go
```

## Files

- `main.go` - Main application file with stub endpoints (you implement the logic here)
- `storage.go` - Optional storage helper for file persistence
- `go.mod` - Go module file

## Tips

- Use `time.Now()` and `time.RFC3339` for timestamps
- Use `github.com/google/uuid` or simple counter for generating IDs
- Go's `encoding/json` package handles JSON marshaling/unmarshaling
- Consider using a slice of structs for queue storage
- Use `sort.Slice()` for maintaining queue order
- Test your endpoints with `curl` before running validation
