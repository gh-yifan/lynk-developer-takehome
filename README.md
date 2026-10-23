# Spacecraft Command Queue - Developer Take-Home Challenge

**Estimated Time:** 2-3 hours

## Background

At Lynk Global, we provide satellite-direct-to-standard-mobile-phone connectivity through a constellation of satellites in Low Earth Orbit. Our ground operations team needs to send commands to spacecraft for various operations like radio configuration, telemetry collection, and system management.

Your task is to build a **Command Queue API** that manages commands waiting to be executed on a spacecraft. Commands must execute in priority order, and the system must track each command's lifecycle through valid state transitions.

## Problem Statement

Build a REST API service that manages a priority-based command queue with the following capabilities:

1. Accept new commands with priority levels
2. Maintain commands in priority order (HIGH → MEDIUM → LOW)
3. Within the same priority, commands execute in FIFO order
4. Enforce valid state transitions as commands are processed
5. Validate command text length
6. Allow deletion of commands from the queue

## Requirements

### Command Priority

Commands have three priority levels:
- **HIGH** - Critical operations (e.g., emergency shutdown)
- **MEDIUM** - Important but not urgent (e.g., configuration updates)
- **LOW** - Routine operations (e.g., telemetry requests)

The queue must always return commands in priority order: HIGH first, then MEDIUM, then LOW.

### Command States

Commands transition through these states:

```
PENDING → EXECUTING → COMPLETED
                   → FAILED
```

**Valid transitions:**
- PENDING → EXECUTING
- EXECUTING → COMPLETED
- EXECUTING → FAILED

**Invalid transitions** (must be rejected):
- PENDING → COMPLETED (can't skip execution)
- PENDING → FAILED (can't fail without executing)
- COMPLETED → anything (terminal state)
- FAILED → anything (terminal state)
- EXECUTING → PENDING (can't go backwards)

### Command Validation

- Command text must be non-empty
- Command text must be ≤ 198 characters (spacecraft packet size limit)

### Queue Ordering

When commands are deleted from the queue, remaining commands must maintain proper priority ordering.

## API Specification

Your API must conform to this format for automated validation.

### Command Object Structure

```json
{
  "id": "string (UUID or sequential ID)",
  "command": "string (the command text)",
  "priority": "HIGH|MEDIUM|LOW",
  "status": "PENDING|EXECUTING|COMPLETED|FAILED",
  "created_at": "ISO 8601 timestamp",
  "updated_at": "ISO 8601 timestamp"
}
```

### Required Endpoints

**Get All Commands (in priority order)**
```
GET /commands

Response: 200 OK
{
  "commands": [<array of command objects>]
}
```

**Create Command**
```
POST /commands

Request Body:
{
  "command": "shutdown satellite radio",
  "priority": "HIGH"
}

Response: 201 Created
{
  "command": {<command object>}
}

Error Response: 400 Bad Request
{
  "error": "Command text exceeds 198 characters"
}
```

**Update Command Status**
```
PATCH /commands/{id}/status

Request Body:
{
  "status": "EXECUTING"
}

Response: 200 OK
{
  "command": {<updated command object>}
}

Error Response: 400 Bad Request
{
  "error": "Invalid state transition: PENDING -> COMPLETED"
}

Error Response: 404 Not Found
{
  "error": "Command not found"
}
```

**Delete Command**
```
DELETE /commands/{id}

Response: 200 OK
{
  "message": "Command deleted"
}

Error Response: 404 Not Found
{
  "error": "Command not found"
}
```

You may implement additional endpoints as needed for your solution.

## What's Provided

Each language scaffold includes:

1. **Basic server setup** - Server initialization with 2 stub endpoints (GET /commands, POST /commands)
2. **Storage helper** - Optional file persistence module (saves queue to JSON file)
3. **Project structure** - Basic files and configuration
4. **Validation script** - Automated test suite to validate your implementation

## What You Must Implement

1. **In-memory command storage** - Design your data structure
2. **Priority ordering logic** - Maintain correct queue order
3. **State transition validation** - Enforce valid state changes
4. **Command validation** - Check text length and required fields
5. **API endpoints** - Complete the stub endpoints and add any others needed
6. **Error handling** - Return appropriate HTTP status codes and error messages

## Storage

The queue should be stored **in-memory** (array, list, map, etc.). When the server restarts, the queue resets - this is acceptable.

**Optional:** A storage helper module is provided in each language if you want the queue to persist across restarts. You can use it or ignore it - it's completely optional.

## Getting Started

1. Choose your language: `python/`, `javascript/`, `rust/`, `java/`, or `go/`
2. Follow the README in that directory for setup instructions
3. Implement the command queue logic
4. Run the validation script to check your solution

```bash
cd python/  # or javascript, rust, java, go
# Follow language-specific README for setup
# Implement your solution
# Run validation: ../validate.py
```

## Validation

Run the validation script from the challenge root directory:

```bash
python3 validate.py [python|javascript|rust|java|go]
```

The script will:
- Start your server
- Run test scenarios
- Check that responses match the specification
- Report pass/fail for each test

Your solution must pass all validation tests.

## Evaluation Criteria

Your solution will be evaluated on:

1. **Correctness (50%)**
   - Queue maintains priority order
   - State transitions enforced correctly
   - Validation rules applied

2. **Code Quality (30%)**
   - Clean, readable code
   - Appropriate data structures
   - Good error handling

3. **API Design (20%)**
   - Follows specification
   - Appropriate HTTP status codes
   - Clear error messages

## Submission

Submit your completed solution with:

1. Your implementation in the chosen language directory
2. A brief `SOLUTION.md` explaining:
   - Your approach to priority ordering
   - How you handle state transitions
   - Any design decisions you made
   - How to run your solution

## Notes

- This challenge focuses on **algorithmic thinking** and **API design**
- Keep it simple - don't over-engineer
- The validation script expects exact API format - follow the specification
- Command execution is simulated via status updates - you don't actually execute commands

Good luck! 🛰️
