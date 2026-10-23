# JavaScript Implementation

This scaffold uses Express for the REST API framework.

## Setup

Due to security scans the javascript files have been changed to .txt files. Just rename them back to .js before proceeding.

1. **Install dependencies:**
   ```bash
   npm install
   ```

2. **Run the server:**
   ```bash
   npm start
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
python3 validate.py javascript
```

## Files

- `server.js` - Main application file with stub endpoints (you implement the logic here)
- `storage.js` - Optional storage helper for file persistence
- `package.json` - Node.js dependencies

## Tips

- Use `Date.now()` or `new Date().toISOString()` for timestamps
- Consider using `crypto.randomUUID()` (Node 14.17+) or a simple counter for IDs
- Test your endpoints with `curl` or Postman before running validation
- Use `Array.prototype.sort()` for maintaining queue order
