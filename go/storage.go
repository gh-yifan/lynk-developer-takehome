// Optional Storage Helper
//
// This package provides simple file-based persistence for the command queue.
// You can use this if you want your queue to survive server restarts.
//
// Usage is completely optional - in-memory storage is acceptable.
//
// Example usage:
//   import "encoding/json"
//
//   // Load queue on startup
//   commands, err := LoadQueue()
//
//   // Save queue after modifications
//   err = SaveQueue(commands)

package main

import (
	"encoding/json"
	"os"
)

const storageFile = "queue_data.json"

// SaveQueue saves the command queue to a JSON file.
func SaveQueue(commands interface{}) error {
	data, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(storageFile, data, 0644)
}

// LoadQueue loads the command queue from a JSON file.
// Returns an empty slice if the file doesn't exist.
func LoadQueue() ([]map[string]interface{}, error) {
	data, err := os.ReadFile(storageFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}

	var commands []map[string]interface{}
	err = json.Unmarshal(data, &commands)
	if err != nil {
		return nil, err
	}

	return commands, nil
}
