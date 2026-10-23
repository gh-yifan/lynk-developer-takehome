package com.lynk.commandqueue;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.*;

/**
 * Spacecraft Command Queue API
 *
 * Implement a priority-based command queue for spacecraft operations.
 * See ../README.md for full requirements.
 */
@SpringBootApplication
@RestController
public class CommandQueueApplication {

    // TODO: Define your data structures here
    // Hint: You'll need classes for Command, Priority, Status, etc.

    public static void main(String[] args) {
        SpringApplication.run(CommandQueueApplication.class, args);
    }

    /**
     * Get all commands in priority order (HIGH -> MEDIUM -> LOW).
     * Within the same priority, return in FIFO order.
     *
     * TODO: Implement this endpoint
     */
    @GetMapping("/commands")
    public ResponseEntity<Map<String, List<Object>>> getCommands() {
        // TODO: Implement - return all commands in priority order
        Map<String, List<Object>> response = new HashMap<>();
        response.put("commands", new ArrayList<>());
        return ResponseEntity.ok(response);
    }

    /**
     * Create a new command in the queue.
     *
     * TODO: Implement this endpoint
     * - Validate command text (non-empty, <= 198 chars)
     * - Validate priority (HIGH, MEDIUM, LOW)
     * - Assign ID, status=PENDING, timestamps
     * - Add to queue
     */
    @PostMapping("/commands")
    public ResponseEntity<Map<String, Object>> createCommand(@RequestBody Map<String, String> request) {
        // TODO: Implement command creation
        Map<String, Object> response = new HashMap<>();
        response.put("command", new HashMap<>());
        return ResponseEntity.status(HttpStatus.CREATED).body(response);
    }

    // TODO: Add more endpoints as needed:
    // - PATCH /commands/{id}/status - Update command status
    // - DELETE /commands/{id} - Delete a command
    // - GET /commands/{id} - Get a specific command (optional)
}
