# Rust Implementation

This scaffold uses Axum for the REST API framework with Tokio for async runtime.

## Setup

1. **Install Rust** (if not already installed):
   ```bash
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   ```

2. **Build and run:**
   ```bash
   cargo run
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
python3 validate.py rust
```

## Files

- `src/main.rs` - Main application file with stub endpoints (you implement the logic here)
- `src/storage.rs` - Optional storage helper for file persistence
- `Cargo.toml` - Rust dependencies

## Tips

- Use `chrono` crate for timestamps if needed (add to Cargo.toml)
- Use `uuid` crate for generating unique IDs if needed (add to Cargo.toml)
- Rust's type system will help catch many errors at compile time
- Consider using `Arc<Mutex<Vec<Command>>>` for shared mutable state
- Test your endpoints with `curl` before running validation
