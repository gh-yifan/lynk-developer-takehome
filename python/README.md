# Python Implementation

This scaffold uses FastAPI for the REST API framework.

## Setup

1. **Create a virtual environment:**
   ```bash
   python3 -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   ```

2. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

3. **Run the server:**
   ```bash
   python main.py
   ```

   The server will start on `http://localhost:8080`

4. **Test it works:**
   ```bash
   curl http://localhost:8080/commands
   ```

## API Documentation

Once running, visit `http://localhost:8080/docs` for interactive API documentation (Swagger UI).

## Running Validation

From the parent directory:
```bash
cd ..
python3 validate.py python
```

## Files

- `main.py` - Main application file with stub endpoints (you implement the logic here)
- `storage.py` - Optional storage helper for file persistence
- `requirements.txt` - Python dependencies

## Tips

- FastAPI automatically validates request/response models when you use Pydantic
- Use Python's built-in `datetime` module for timestamps
- Consider using `uuid` module for generating unique IDs
- Test your endpoints with `curl` or the Swagger UI before running validation
