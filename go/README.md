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
