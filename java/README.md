# Java Implementation

This scaffold uses Spring Boot for the REST API framework.

## Setup

1. **Install Java 17 or higher** (if not already installed):
   - Check version: `java -version`
   - Download from: https://adoptium.net/

2. **Install Maven** (if not already installed):
   - Check version: `mvn -version`
   - Download from: https://maven.apache.org/download.cgi

3. **Build and run:**
   ```bash
   mvn spring-boot:run
   ```

   The server will start on `http://localhost:8080`

4. **Test it works:**
   ```bash
   curl http://localhost:8080/commands
   ```

## Running Validation

From the parent directory:
```bash
cd ..
python3 validate.py java
```

## Files

- `src/main/java/com/lynk/commandqueue/CommandQueueApplication.java` - Main application with stub endpoints
- `src/main/java/com/lynk/commandqueue/Storage.java` - Optional storage helper for file persistence
- `pom.xml` - Maven dependencies
- `src/main/resources/application.properties` - Application configuration

## Tips

- Use `java.time.Instant` or `java.time.LocalDateTime` for timestamps
- Use `java.util.UUID.randomUUID()` for generating unique IDs
- Spring Boot's Jackson integration handles JSON serialization automatically
- Consider using `ArrayList` or `LinkedList` for queue storage
- Test your endpoints with `curl` or Postman before running validation
