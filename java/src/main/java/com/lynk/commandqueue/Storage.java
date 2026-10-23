package com.lynk.commandqueue;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;

import java.io.File;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

/**
 * Optional Storage Helper
 *
 * This class provides simple file-based persistence for the command queue.
 * You can use this if you want your queue to survive server restarts.
 *
 * Usage is completely optional - in-memory storage is acceptable.
 *
 * Example usage:
 *   Storage storage = new Storage();
 *
 *   // Load queue on startup
 *   List<Command> commands = storage.loadQueue(Command.class);
 *
 *   // Save queue after modifications
 *   storage.saveQueue(commands);
 */
public class Storage {
    private static final String STORAGE_FILE = "queue_data.json";
    private final ObjectMapper objectMapper;

    public Storage() {
        this.objectMapper = new ObjectMapper();
        this.objectMapper.enable(SerializationFeature.INDENT_OUTPUT);
    }

    /**
     * Save the command queue to a JSON file.
     *
     * @param commands List of command objects
     * @throws IOException if file writing fails
     */
    public void saveQueue(List<?> commands) throws IOException {
        objectMapper.writeValue(new File(STORAGE_FILE), commands);
    }

    /**
     * Load the command queue from a JSON file.
     *
     * @param clazz The class type to deserialize into
     * @return List of command objects, or empty list if file doesn't exist
     * @throws IOException if file reading fails (but not for missing file)
     */
    public <T> List<T> loadQueue(Class<T> clazz) throws IOException {
        File file = new File(STORAGE_FILE);
        if (!file.exists()) {
            return new ArrayList<>();
        }
        return objectMapper.readValue(
            file,
            objectMapper.getTypeFactory().constructCollectionType(List.class, clazz)
        );
    }
}
