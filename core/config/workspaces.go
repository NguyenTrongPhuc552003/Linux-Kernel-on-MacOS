// Package config provides workspace management for elmos.
// Workspaces are isolated directories for building specific boards.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// WorkspaceManager handles workspace-related operations.
type WorkspaceManager struct {
	rootPath string // Path to workspace root (where .elmos/ exists)
}

// NewWorkspaceManager creates a new WorkspaceManager.
// rootPath should point to the workspace root directory.
func NewWorkspaceManager(rootPath string) *WorkspaceManager {
	if rootPath == "" {
		rootPath = "."
	}
	return &WorkspaceManager{
		rootPath: rootPath,
	}
}

// FindWorkspaceRoot searches for .elmos/ directory starting from cwd and going up.
// Returns the workspace root path or error if not found.
func FindWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search up the directory tree for .elmos/
	for {
		elmosPath := filepath.Join(cwd, ".elmos")
		if info, err := os.Stat(elmosPath); err == nil && info.IsDir() {
			return cwd, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached root directory
			break
		}
		cwd = parent
	}

	return "", fmt.Errorf("workspace root (.elmos/) not found in current directory or any parent")
}

// GetRoot returns the workspace root path.
func (wm *WorkspaceManager) GetRoot() string {
	return wm.rootPath
}

// GetElmosDir returns the .elmos/ directory path.
func (wm *WorkspaceManager) GetElmosDir() string {
	return filepath.Join(wm.rootPath, ".elmos")
}

// GetConfigDir returns the .elmos/config directory path.
func (wm *WorkspaceManager) GetConfigDir() string {
	return filepath.Join(wm.GetElmosDir(), "config")
}

// GetCacheDir returns the .elmos/build-cache directory path.
func (wm *WorkspaceManager) GetCacheDir() string {
	return filepath.Join(wm.GetElmosDir(), "build-cache")
}

// GetMachineDir returns the directory for machine definitions.
func (wm *WorkspaceManager) GetMachineDir() string {
	return filepath.Join(wm.rootPath, "machine")
}

// GetMachineDefPath returns the path to a specific machine definition file.
func (wm *WorkspaceManager) GetMachineDefPath(machineID string) string {
	return filepath.Join(wm.GetMachineDir(), fmt.Sprintf("%s.yml", machineID))
}

// GetPluginsDir returns the directory for user plugins.
func (wm *WorkspaceManager) GetPluginsDir() string {
	return filepath.Join(wm.rootPath, "plugins")
}

// GetDefaultConfigPath returns the path to default.yml.
func (wm *WorkspaceManager) GetDefaultConfigPath() string {
	return filepath.Join(wm.rootPath, "default.yml")
}

// GetPluginConfigPath returns the path to .elmos/plugins.yml.
func (wm *WorkspaceManager) GetPluginConfigPath() string {
	return filepath.Join(wm.GetElmosDir(), "plugins.yml")
}

// Initialize creates the workspace directory structure.
// Creates .elmos/, machine/, plugins/ directories if they don't exist.
func (wm *WorkspaceManager) Initialize() error {
	dirs := []string{
		wm.GetElmosDir(),
		wm.GetConfigDir(),
		wm.GetCacheDir(),
		wm.GetMachineDir(),
		wm.GetPluginsDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// Exists checks if the workspace is initialized (has .elmos/ directory).
func (wm *WorkspaceManager) Exists() bool {
	elmosDir := wm.GetElmosDir()
	info, err := os.Stat(elmosDir)
	return err == nil && info.IsDir()
}

// EnsureInitialized verifies the workspace is properly initialized.
// Returns error if critical directories are missing.
func (wm *WorkspaceManager) EnsureInitialized() error {
	if !wm.Exists() {
		return fmt.Errorf("workspace not initialized at %s", wm.rootPath)
	}

	// Check for critical subdirs
	requiredDirs := []string{
		wm.GetElmosDir(),
		wm.GetCacheDir(),
	}

	for _, dir := range requiredDirs {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return fmt.Errorf("workspace directory missing or invalid: %s", dir)
		}
	}

	return nil
}

// CleanCache deletes the build cache directory.
// Useful for forcing a clean rebuild.
func (wm *WorkspaceManager) CleanCache() error {
	cacheDir := wm.GetCacheDir()
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to clean cache: %w", err)
	}

	// Recreate the empty cache directory
	return os.MkdirAll(cacheDir, 0755)
}

// GetCachePath returns the path for a specific cache artifact.
func (wm *WorkspaceManager) GetCachePath(fingerprint string) string {
	return filepath.Join(wm.GetCacheDir(), fingerprint)
}

// GetCacheIndexPath returns the path to the build cache index file.
func (wm *WorkspaceManager) GetCacheIndexPath() string {
	return filepath.Join(wm.GetCacheDir(), "index.json")
}

// QueryMachines returns all machine definitions in the workspace.
func (wm *WorkspaceManager) QueryMachines() ([]string, error) {
	machineDir := wm.GetMachineDir()

	// Return empty list if machine directory doesn't exist yet
	if _, err := os.Stat(machineDir); err != nil {
		return []string{}, nil
	}

	entries, err := os.ReadDir(machineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read machine directory: %w", err)
	}

	var machines []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yml" {
			name := entry.Name()[:len(entry.Name())-4] // Remove .yml
			machines = append(machines, name)
		}
	}

	return machines, nil
}
