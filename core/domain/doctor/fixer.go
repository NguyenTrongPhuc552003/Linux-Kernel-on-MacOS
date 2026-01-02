// Package doctor provides dependency checking and environment validation for elmos.
package doctor

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
)

// AutoFixer provides methods to automatically fix common issues.
type AutoFixer struct {
	fs  filesystem.FileSystem
	cfg *elconfig.Config
}

// NewAutoFixer creates a new AutoFixer with the given dependencies.
func NewAutoFixer(fs filesystem.FileSystem, cfg *elconfig.Config) *AutoFixer {
	return &AutoFixer{
		fs:  fs,
		cfg: cfg,
	}
}

// FixElfH downloads elf.h from glibc if it's missing.
func (f *AutoFixer) FixElfH() error {
	headersDir := f.cfg.Paths.LibrariesDir
	elfPath := filepath.Join(headersDir, "elf.h")

	// Check if it already exists
	if f.fs.Exists(elfPath) {
		return nil
	}

	// Ensure directory exists
	if err := f.fs.MkdirAll(headersDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Download from glibc
	url := fmt.Sprintf("https://raw.githubusercontent.com/bminor/glibc/glibc-%s/elf/elf.h",
		elconfig.DefaultGlibcVersion)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := f.fs.WriteFile(elfPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// CanFixElfH checks if elf.h can be downloaded (is missing).
func (f *AutoFixer) CanFixElfH() bool {
	elfPath := filepath.Join(f.cfg.Paths.LibrariesDir, "elf.h")
	return !f.fs.Exists(elfPath)
}
