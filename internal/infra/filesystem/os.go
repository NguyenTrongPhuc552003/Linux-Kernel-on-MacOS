// Package filesystem provides abstractions for file system operations.
package filesystem

import (
	"io/fs"
	"os"
)

// OSFileSystem implements FileSystem using the standard os package.
type OSFileSystem struct{}

// NewOSFileSystem creates a new OSFileSystem.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// Stat returns file info for the named file.
func (f *OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile reads the entire contents of the named file.
func (f *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFile writes data to the named file, creating it if necessary.
func (f *OSFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// MkdirAll creates a directory and all necessary parents.
func (f *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// ReadDir reads the named directory and returns its entries.
func (f *OSFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// Remove removes the named file or empty directory.
func (f *OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

// RemoveAll removes the path and any children it contains.
func (f *OSFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Exists returns true if the path exists.
func (f *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir returns true if the path is a directory.
func (f *OSFileSystem) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Getwd returns the current working directory.
func (f *OSFileSystem) Getwd() (string, error) {
	return os.Getwd()
}

// Create creates or truncates the named file.
func (f *OSFileSystem) Create(name string) (*os.File, error) {
	return os.Create(name)
}

// Open opens the named file for reading.
func (f *OSFileSystem) Open(name string) (*os.File, error) {
	return os.Open(name)
}

// Ensure OSFileSystem implements FileSystem.
var _ FileSystem = (*OSFileSystem)(nil)
