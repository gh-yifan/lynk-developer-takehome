#!/usr/bin/env python3
"""
Command Queue API Validation Script

This script validates that a candidate's implementation meets the requirements.
It starts the server, runs test scenarios, and reports pass/fail results.

Usage:
    python3 validate.py [python|javascript|rust|java|go]
"""

import sys
import subprocess
import time
import requests
import json
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from threading import Barrier
from typing import Dict, List, Optional, Tuple
import os
import signal
import uuid

BASE_URL = "http://localhost:8080"

class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    END = '\033[0m'

class ServerManager:
    """Manages starting and stopping the server process."""

    def __init__(self, language: str):
        self.language = language
        self.process = None

    def start(self) -> bool:
        """Start the server based on language."""
        try:
            if self.language == "python":
                os.chdir("python")
                self.process = subprocess.Popen(
                    ["python3", "main.py"],
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE
                )
            elif self.language == "javascript":
                os.chdir("javascript")
                self.process = subprocess.Popen(
                    ["node", "server.js"],
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE
                )
            elif self.language == "rust":
                os.chdir("rust")
                self.process = subprocess.Popen(
                    ["cargo", "run", "--release"],
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE
                )
            elif self.language == "java":
                os.chdir("java")
                self.process = subprocess.Popen(
                    ["mvn", "spring-boot:run"],
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE
                )
            elif self.language == "go":
                os.chdir("go")
                self.process = subprocess.Popen(
                    ["go", "run", "-race", "main.go", "storage.go"],
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE
                )
            else:
                print(f"{Colors.RED}Unknown language: {self.language}{Colors.END}")
                return False

            print(f"{Colors.BLUE}Starting {self.language} server...{Colors.END}")
            return self._wait_for_server()

        except Exception as e:
            print(f"{Colors.RED}Failed to start server: {e}{Colors.END}")
            return False

    def _wait_for_server(self, timeout: int = 90) -> bool:
        """Wait for server to be ready."""
        start = time.time()
        while time.time() - start < timeout:
            if self.process.poll() is not None:
                stdout, stderr = self.process.communicate()
                print(f"{Colors.RED}Server process exited with code {self.process.returncode}.{Colors.END}")
                if stdout:
                    print(stdout.decode(errors="replace"))
                if stderr:
                    print(stderr.decode(errors="replace"))
                return False

            try:
                response = requests.get(f"{BASE_URL}/commands", timeout=1)
                print(f"{Colors.GREEN}Server is ready!{Colors.END}")
                return True
            except requests.exceptions.RequestException:
                time.sleep(1)

        print(f"{Colors.RED}Server failed to start within {timeout} seconds{Colors.END}")
        self.process.terminate()
        try:
            stdout, stderr = self.process.communicate(timeout=5)
        except subprocess.TimeoutExpired:
            self.process.kill()
            stdout, stderr = self.process.communicate()
        if stdout:
            print(stdout.decode(errors="replace"))
        if stderr:
            print(stderr.decode(errors="replace"))
        return False

    def stop(self):
        """Stop the server process."""
        if self.process:
            print(f"{Colors.BLUE}Stopping server...{Colors.END}")
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()

class TestRunner:
    """Runs validation tests against the API."""

    def __init__(self):
        self.passed = 0
        self.failed = 0
        self.tests = []

    def test(self, name: str, fn):
        """Run a test and record result."""
        try:
            fn()
            self.passed += 1
            print(f"{Colors.GREEN}✓{Colors.END} Test {self.passed + self.failed}: {name}")
            return True
        except AssertionError as e:
            self.failed += 1
            print(f"{Colors.RED}✗{Colors.END} Test {self.passed + self.failed}: {name}")
            print(f"  {Colors.RED}Error: {e}{Colors.END}")
            return False
        except Exception as e:
            self.failed += 1
            print(f"{Colors.RED}✗{Colors.END} Test {self.passed + self.failed}: {name}")
            print(f"  {Colors.RED}Unexpected error: {e}{Colors.END}")
            return False

    def report(self):
        """Print final test report."""
        total = self.passed + self.failed
        percentage = (self.passed / total * 100) if total > 0 else 0

        print(f"\n{'='*60}")
        print(f"Test Results: {self.passed}/{total} passed ({percentage:.1f}%)")
        print(f"{'='*60}\n")

        if self.failed == 0:
            print(f"{Colors.GREEN}All tests passed! 🎉{Colors.END}")
        else:
            print(f"{Colors.YELLOW}{self.failed} test(s) failed. Review the output above.{Colors.END}")

def test_create_command_high_priority(runner: TestRunner):
    """Test creating a command with HIGH priority."""
    def test():
        response = requests.post(
            f"{BASE_URL}/commands",
            json={"command": "shutdown radio", "priority": "HIGH"}
        )
        assert response.status_code == 201, f"Expected 201, got {response.status_code}"
        data = response.json()
        assert "command" in data, "Response missing 'command' field"
        cmd = data["command"]
        assert cmd.get("priority") == "HIGH", f"Expected priority HIGH, got {cmd.get('priority')}"
        assert cmd.get("status") == "PENDING", f"Expected status PENDING, got {cmd.get('status')}"
        assert "id" in cmd, "Command missing 'id' field"
        assert "created_at" in cmd, "Command missing 'created_at' field"
        assert "updated_at" in cmd, "Command missing 'updated_at' field"

    runner.test("Create command with HIGH priority", test)

def test_create_command_validation(runner: TestRunner):
    """Test command validation (empty text)."""
    def test():
        response = requests.post(
            f"{BASE_URL}/commands",
            json={"command": "", "priority": "HIGH"}
        )
        assert response.status_code == 400, f"Expected 400 for empty command, got {response.status_code}"
        data = response.json()
        assert "error" in data, "Error response should contain 'error' field"

    runner.test("Reject empty command text", test)

def test_create_command_max_length(runner: TestRunner):
    """Test command validation (>198 chars)."""
    def test():
        long_command = "x" * 199
        response = requests.post(
            f"{BASE_URL}/commands",
            json={"command": long_command, "priority": "MEDIUM"}
        )
        assert response.status_code == 400, f"Expected 400 for command >198 chars, got {response.status_code}"
        data = response.json()
        assert "error" in data, "Error response should contain 'error' field"

    runner.test("Reject command text >198 characters", test)

def test_command_validation_boundaries(runner: TestRunner):
    """Whitespace is trimmed and the 198-character boundary is accepted."""
    def test():
        response = requests.post(
            f"{BASE_URL}/commands",
            json={"command": " " + ("x" * 198) + " ", "priority": "HIGH"}
        )
        assert response.status_code == 201, f"Expected 201 for trimmed 198-character command, got {response.status_code}"
        command = response.json()["command"]["command"]
        assert command == "x" * 198, "Command should be trimmed and retain exactly 198 characters"

        response = requests.post(
            f"{BASE_URL}/commands",
            json={"command": " \t\n ", "priority": "HIGH"}
        )
        assert response.status_code == 400, f"Expected 400 for whitespace-only command, got {response.status_code}"

        response = requests.post(
            f"{BASE_URL}/commands",
            json={"command": "invalid priority", "priority": "URGENT"}
        )
        assert response.status_code == 400, f"Expected 400 for invalid priority, got {response.status_code}"

    runner.test("Extra: command validation boundaries", test)

def test_priority_ordering(runner: TestRunner):
    """Test that commands are returned in priority order."""
    def test():
        # Clear any existing commands first
        requests.post(f"{BASE_URL}/commands", json={"command": "low task", "priority": "LOW"})
        requests.post(f"{BASE_URL}/commands", json={"command": "high task", "priority": "HIGH"})
        requests.post(f"{BASE_URL}/commands", json={"command": "medium task", "priority": "MEDIUM"})

        response = requests.get(f"{BASE_URL}/commands")
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        data = response.json()
        assert "commands" in data, "Response missing 'commands' field"

        commands = data["commands"]
        assert len(commands) >= 3, f"Expected at least 3 commands, got {len(commands)}"

        # Find our test commands
        priorities = [cmd["priority"] for cmd in commands if "task" in cmd.get("command", "")]

        # HIGH should come before MEDIUM and LOW
        high_idx = priorities.index("HIGH") if "HIGH" in priorities else -1
        medium_idx = priorities.index("MEDIUM") if "MEDIUM" in priorities else -1
        low_idx = priorities.index("LOW") if "LOW" in priorities else -1

        assert high_idx < medium_idx, f"HIGH priority should come before MEDIUM"
        assert medium_idx < low_idx, f"MEDIUM priority should come before LOW"

    runner.test("Commands returned in priority order (HIGH -> MEDIUM -> LOW)", test)

def test_fifo_within_priority(runner: TestRunner):
    """Test FIFO ordering within same priority."""
    def test():
        # Create multiple HIGH priority commands
        response1 = requests.post(f"{BASE_URL}/commands", json={"command": "first high", "priority": "HIGH"})
        id1 = response1.json()["command"]["id"]

        time.sleep(0.1)  # Small delay to ensure different timestamps

        response2 = requests.post(f"{BASE_URL}/commands", json={"command": "second high", "priority": "HIGH"})
        id2 = response2.json()["command"]["id"]

        response = requests.get(f"{BASE_URL}/commands")
        commands = response.json()["commands"]

        # Find our commands
        cmd1_idx = next((i for i, cmd in enumerate(commands) if cmd["id"] == id1), -1)
        cmd2_idx = next((i for i, cmd in enumerate(commands) if cmd["id"] == id2), -1)

        assert cmd1_idx >= 0 and cmd2_idx >= 0, "Commands not found in queue"
        assert cmd1_idx < cmd2_idx, f"First command should come before second (FIFO)"

    runner.test("FIFO ordering within same priority", test)

def test_valid_state_transition(runner: TestRunner):
    """Test valid state transition (PENDING -> EXECUTING)."""
    def test():
        # Create a command
        response = requests.post(f"{BASE_URL}/commands", json={"command": "test cmd", "priority": "MEDIUM"})
        cmd_id = response.json()["command"]["id"]

        # Update status to EXECUTING
        response = requests.patch(
            f"{BASE_URL}/commands/{cmd_id}/status",
            json={"status": "EXECUTING"}
        )
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        data = response.json()
        assert data["command"]["status"] == "EXECUTING", f"Status should be EXECUTING"

    runner.test("Valid state transition: PENDING -> EXECUTING", test)

def test_valid_state_transition_to_completed(runner: TestRunner):
    """Test valid state transition (EXECUTING -> COMPLETED)."""
    def test():
        # Create and move to EXECUTING
        response = requests.post(f"{BASE_URL}/commands", json={"command": "complete me", "priority": "LOW"})
        cmd_id = response.json()["command"]["id"]
        requests.patch(f"{BASE_URL}/commands/{cmd_id}/status", json={"status": "EXECUTING"})

        # Move to COMPLETED
        response = requests.patch(
            f"{BASE_URL}/commands/{cmd_id}/status",
            json={"status": "COMPLETED"}
        )
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        assert response.json()["command"]["status"] == "COMPLETED"

    runner.test("Valid state transition: EXECUTING -> COMPLETED", test)

def test_invalid_state_transition(runner: TestRunner):
    """Test invalid state transition (PENDING -> COMPLETED)."""
    def test():
        # Create a command
        response = requests.post(f"{BASE_URL}/commands", json={"command": "bad transition", "priority": "HIGH"})
        cmd_id = response.json()["command"]["id"]

        # Try invalid transition
        response = requests.patch(
            f"{BASE_URL}/commands/{cmd_id}/status",
            json={"status": "COMPLETED"}
        )
        assert response.status_code == 400, f"Expected 400 for invalid transition, got {response.status_code}"
        data = response.json()
        assert "error" in data, "Error response should contain 'error' field"
        assert "transition" in data["error"].lower(), "Error should mention invalid transition"

    runner.test("Invalid state transition rejected: PENDING -> COMPLETED", test)

def test_delete_command(runner: TestRunner):
    """Test deleting a command."""
    def test():
        # Create a command
        response = requests.post(f"{BASE_URL}/commands", json={"command": "delete me", "priority": "LOW"})
        cmd_id = response.json()["command"]["id"]

        # Delete it
        response = requests.delete(f"{BASE_URL}/commands/{cmd_id}")
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"

        # Verify it's gone
        response = requests.get(f"{BASE_URL}/commands")
        commands = response.json()["commands"]
        assert not any(cmd["id"] == cmd_id for cmd in commands), "Deleted command still in queue"

    runner.test("Delete command removes it from queue", test)

def test_delete_nonexistent_command(runner: TestRunner):
    """Test deleting a non-existent command."""
    def test():
        response = requests.delete(f"{BASE_URL}/commands/nonexistent-id-12345")
        assert response.status_code == 404, f"Expected 404 for non-existent command, got {response.status_code}"
        data = response.json()
        assert "error" in data, "Error response should contain 'error' field"

    runner.test("Delete non-existent command returns 404", test)

# Additional regression tests, kept after the original validator cases.
def test_valid_state_transition_to_failed(runner: TestRunner):
    """Test valid state transition (EXECUTING -> FAILED)."""
    def test():
        response = requests.post(f"{BASE_URL}/commands", json={"command": "fail me", "priority": "LOW"})
        assert response.status_code == 201, f"Expected 201, got {response.status_code}"
        cmd_id = response.json()["command"]["id"]
        response = requests.patch(f"{BASE_URL}/commands/{cmd_id}/status", json={"status": "EXECUTING"})
        assert response.status_code == 200, f"Expected 200 entering EXECUTING, got {response.status_code}"
        response = requests.patch(f"{BASE_URL}/commands/{cmd_id}/status", json={"status": "FAILED"})
        assert response.status_code == 200, f"Expected 200 entering FAILED, got {response.status_code}"
        assert response.json()["command"]["status"] == "FAILED"

    runner.test("Extra: EXECUTING -> FAILED", test)

def test_get_reflects_patch(runner: TestRunner):
    """Ensure GET returns the patched status and updated timestamp."""
    def test():
        response = requests.post(f"{BASE_URL}/commands", json={"command": "check GET after PATCH", "priority": "HIGH"})
        assert response.status_code == 201, f"Expected 201, got {response.status_code}"
        created = response.json()["command"]
        time.sleep(0.02)
        response = requests.patch(f"{BASE_URL}/commands/{created['id']}/status", json={"status": "EXECUTING"})
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        patched = response.json()["command"]
        response = requests.get(f"{BASE_URL}/commands")
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        fetched = next((cmd for cmd in response.json()["commands"] if cmd["id"] == created["id"]), None)
        assert fetched is not None, "Patched command missing from GET /commands"
        assert fetched["status"] == "EXECUTING", f"GET returned stale status: {fetched['status']}"
        created_at = datetime.fromisoformat(created["created_at"].replace("Z", "+00:00"))
        updated_at = datetime.fromisoformat(fetched["updated_at"].replace("Z", "+00:00"))
        assert updated_at > created_at, "updated_at did not advance after PATCH"
        assert fetched["updated_at"] == patched["updated_at"], "GET and PATCH returned different updated_at values"

    runner.test("Extra: GET reflects PATCH status and updated_at", test)

def test_competing_terminal_transitions(runner: TestRunner):
    """Only one of two concurrent terminal transitions from EXECUTING may succeed."""
    def test():
        for iteration in range(10):
            response = requests.post(f"{BASE_URL}/commands", json={"command": f"competing-transition-{iteration}", "priority": "HIGH"})
            assert response.status_code == 201, f"Expected 201, got {response.status_code}"
            cmd_id = response.json()["command"]["id"]
            response = requests.patch(f"{BASE_URL}/commands/{cmd_id}/status", json={"status": "EXECUTING"})
            assert response.status_code == 200, f"Expected 200 entering EXECUTING, got {response.status_code}"

            start = Barrier(3)

            def patch_terminal(status):
                start.wait()
                return requests.patch(f"{BASE_URL}/commands/{cmd_id}/status", json={"status": status}, timeout=10)

            with ThreadPoolExecutor(max_workers=2) as executor:
                completed = executor.submit(patch_terminal, "COMPLETED")
                failed = executor.submit(patch_terminal, "FAILED")
                start.wait()
                responses = [completed.result(), failed.result()]

            statuses = sorted(response.status_code for response in responses)
            response = requests.get(f"{BASE_URL}/commands")
            assert response.status_code == 200, f"GET after competing PATCH returned {response.status_code}"
            command = next(
                (item for item in response.json()["commands"] if item["id"] == cmd_id),
                None,
            )
            final_status = command["status"] if command else "<missing>"
            details = [
                {"http_status": item.status_code, "body": item.text}
                for item in responses
            ]
            assert statuses == [200, 400], (
                f"Expected one 200 and one 400, got {statuses}; "
                f"PATCH responses={details}; final stored status={final_status}"
            )

    runner.test("Extra: competing terminal PATCH requests", test)

def test_delete_queue_positions(runner: TestRunner):
    """Deleting first, middle, and last entries preserves FIFO order."""
    def test():
        run_id = uuid.uuid4().hex
        for target_index in (0, 2, 3):
            created = []
            for i in range(4):
                command_text = f"delete-{run_id}-{target_index}-{i}"
                response = requests.post(f"{BASE_URL}/commands", json={"command": command_text, "priority": "MEDIUM"})
                assert response.status_code == 201, f"Expected 201, got {response.status_code}"
                created.append(response.json()["command"])

            target = created[target_index]
            response = requests.delete(f"{BASE_URL}/commands/{target['id']}")
            assert response.status_code == 200, f"Delete returned {response.status_code}"
            response = requests.get(f"{BASE_URL}/commands")
            remaining = [cmd["id"] for cmd in response.json()["commands"] if cmd.get("command", "").startswith(f"delete-{run_id}-{target_index}-")]
            expected = [cmd["id"] for i, cmd in enumerate(created) if i != target_index]
            assert remaining == expected, f"FIFO after deleting position {target_index}: got {remaining}, want {expected}"

    runner.test("Extra: delete at first, middle, and last positions", test)

def test_concurrent_creates(runner: TestRunner):
    """Concurrent POSTs must all be retained; Go runs with the race detector."""
    def test():
        count = 32
        run_id = uuid.uuid4().hex

        def create(index):
            return requests.post(f"{BASE_URL}/commands", json={"command": f"concurrent-create-{run_id}-{index}", "priority": "LOW"}, timeout=10)

        with ThreadPoolExecutor(max_workers=count) as executor:
            responses = list(executor.map(create, range(count)))
        for response in responses:
            assert response.status_code == 201, f"Concurrent POST returned {response.status_code}: {response.text}"

        response = requests.get(f"{BASE_URL}/commands")
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        commands = [cmd for cmd in response.json()["commands"] if cmd.get("command", "").startswith(f"concurrent-create-{run_id}-")]
        assert len(commands) == count, f"Retained {len(commands)} of {count} concurrent creates"
        ids = [cmd["id"] for cmd in commands]
        assert len(set(ids)) == count, "Concurrent creates produced duplicate IDs"

    runner.test("Extra: concurrent command creation", test)

def test_rejected_transitions_preserve_status(runner: TestRunner):
    """Invalid transitions must fail without changing the stored status."""
    def test():
        cases = [
            ("pending-to-failed", "PENDING", "FAILED"),
            ("executing-to-pending", "EXECUTING", "PENDING"),
            ("completed-is-terminal", "COMPLETED", "FAILED"),
            ("failed-is-terminal", "FAILED", "COMPLETED"),
        ]

        for label, initial_status, requested_status in cases:
            response = requests.post(
                f"{BASE_URL}/commands",
                json={"command": f"transition-regression-{label}", "priority": "MEDIUM"}
            )
            assert response.status_code == 201, f"Create failed for {label}: {response.status_code}"
            command = response.json()["command"]
            command_id = command["id"]

            if initial_status != "PENDING":
                response = requests.patch(
                    f"{BASE_URL}/commands/{command_id}/status",
                    json={"status": "EXECUTING"}
                )
                assert response.status_code == 200, f"Could not enter EXECUTING for {label}: {response.status_code}"
            if initial_status in ("COMPLETED", "FAILED"):
                response = requests.patch(
                    f"{BASE_URL}/commands/{command_id}/status",
                    json={"status": initial_status}
                )
                assert response.status_code == 200, f"Could not enter {initial_status} for {label}: {response.status_code}"

            response = requests.patch(
                f"{BASE_URL}/commands/{command_id}/status",
                json={"status": requested_status}
            )
            assert response.status_code == 400, f"Expected 400 for {label}, got {response.status_code}"

            response = requests.get(f"{BASE_URL}/commands")
            assert response.status_code == 200, f"GET after rejected transition returned {response.status_code}"
            stored = next(
                (item for item in response.json()["commands"] if item["id"] == command_id),
                None,
            )
            assert stored is not None, f"Command missing after rejected transition: {label}"
            assert stored["status"] == initial_status, (
                f"Rejected transition {label} changed status to {stored['status']}"
            )

    runner.test("Extra: rejected transitions preserve status", test)

def main():
    if len(sys.argv) != 2:
        print("Usage: python3 validate.py [python|javascript|rust|java|go]")
        sys.exit(1)

    language = sys.argv[1]

    if language not in ["python", "javascript", "rust", "java", "go"]:
        print(f"{Colors.RED}Invalid language: {language}{Colors.END}")
        print("Must be one of: python, javascript, rust, java, go")
        sys.exit(1)

    print(f"\n{'='*60}")
    print(f"Command Queue API Validation - {language.upper()}")
    print(f"{'='*60}\n")

    server = ServerManager(language)

    try:
        if not server.start():
            print(f"{Colors.RED}Failed to start server. Exiting.{Colors.END}")
            sys.exit(1)

        print(f"\n{Colors.BLUE}Running validation tests...{Colors.END}\n")

        runner = TestRunner()

        # Run all tests
        test_create_command_high_priority(runner)
        test_create_command_validation(runner)
        test_create_command_max_length(runner)
        test_command_validation_boundaries(runner)
        test_priority_ordering(runner)
        test_fifo_within_priority(runner)
        test_valid_state_transition(runner)
        test_valid_state_transition_to_completed(runner)
        test_invalid_state_transition(runner)
        test_delete_command(runner)
        test_delete_nonexistent_command(runner)

        print(f"\n{Colors.BLUE}Additional regression tests...{Colors.END}\n")
        test_valid_state_transition_to_failed(runner)
        test_get_reflects_patch(runner)
        test_competing_terminal_transitions(runner)
        test_delete_queue_positions(runner)
        test_concurrent_creates(runner)
        test_rejected_transitions_preserve_status(runner)

        runner.report()

        sys.exit(0 if runner.failed == 0 else 1)

    finally:
        server.stop()

if __name__ == "__main__":
    main()
