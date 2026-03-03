// Package executor provides abstractions for executing shell commands.
// This file contains environment variable merging utilities.
package executor

import (
	"os"
	"strings"
)

// mergeEnv merges environment variables, with override taking precedence over base.
// This prevents duplicate environment variable keys and ensures consistent behavior.
// Format: "KEY=value"
//
// Example:
//
//	base := os.Environ()  // includes PATH=/usr/bin
//	override := []string{"PATH=/custom/bin"}
//	result := mergeEnv(base, override)  // PATH=/custom/bin (override wins)
func mergeEnv(base, override []string) []string {
	// Build a map from override first (higher priority)
	envMap := make(map[string]string, len(base)+len(override))

	// Add base environment
	for _, entry := range base {
		key, value := splitEnv(entry)
		if key != "" {
			envMap[key] = value
		}
	}

	// Override with custom environment
	for _, entry := range override {
		key, value := splitEnv(entry)
		if key != "" {
			envMap[key] = value
		}
	}

	// Convert map back to slice
	result := make([]string, 0, len(envMap))
	for key, value := range envMap {
		result = append(result, key+"="+value)
	}

	return result
}

// splitEnv splits an environment variable entry into key and value.
// Returns ("", "") for invalid entries.
func splitEnv(entry string) (key, value string) {
	idx := strings.IndexByte(entry, '=')
	if idx == -1 || idx == 0 {
		return "", ""
	}
	return entry[:idx], entry[idx+1:]
}

// GetCurrentEnv returns the current process environment as a map.
// Useful for debugging and testing.
func GetCurrentEnv() map[string]string {
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value := splitEnv(entry)
		if key != "" {
			env[key] = value
		}
	}
	return env
}
