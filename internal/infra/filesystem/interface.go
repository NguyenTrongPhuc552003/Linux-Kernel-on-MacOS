// Package filesystem provides abstractions for file system operations.
package filesystem

import (
	"io/fs"
	"os"
)

// FileSystem defines the interface for file system operations.
// This abstraction allows for easy mocking in tests and potential
// future implementations (e.g., in-memory FS, remote FS).
type FileSystem interface {
	// Stat returns file info for the named file.
	Stat(name string) (os.FileInfo, error)

	// ReadFile reads the entire contents of the named file.
	ReadFile(name string) ([]byte, error)

	// WriteFile writes data to the named file, creating it if necessary.
	WriteFile(name string, data []byte, perm os.FileMode) error

	// MkdirAll creates a directory and all necessary parents.
	MkdirAll(path string, perm os.FileMode) error

	// ReadDir reads the named directory and returns its entries.
	ReadDir(name string) ([]fs.DirEntry, error)

	// Remove removes the named file or empty directory.
	Remove(name string) error

	// RemoveAll removes the path and any children it contains.
	RemoveAll(path string) error

	// Exists returns true if the path exists.
	Exists(path string) bool

	// IsDir returns true if the path is a directory.
	IsDir(path string) bool

	// Getwd returns the current working directory.
	Getwd() (string, error)

	// Create creates or truncates the named file.
	Create(name string) (*os.File, error)

	// Open opens the named file for reading.
	Open(name string) (*os.File, error)
}
