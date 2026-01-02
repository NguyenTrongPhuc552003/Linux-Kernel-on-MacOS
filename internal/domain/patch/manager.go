// Package patch provides kernel patch management for elmos.
package patch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/internal/config"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
)

// PatchInfo contains information about a patch file.
type PatchInfo struct {
	Name    string
	Path    string
	Version string
}

// Manager handles kernel patch operations.
type Manager struct {
	exec executor.Executor
	fs   filesystem.FileSystem
	cfg  *elconfig.Config
}

// NewManager creates a new patch Manager.
func NewManager(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config) *Manager {
	return &Manager{
		exec: exec,
		fs:   fs,
		cfg:  cfg,
	}
}

// Apply applies a patch file to the kernel source.
func (m *Manager) Apply(ctx context.Context, patchFile string) error {
	// Resolve patch path
	patchPath := patchFile
	if !filepath.IsAbs(patchPath) {
		// Check in patches directory first
		testPath := filepath.Join(m.cfg.Paths.PatchesDir, patchPath)
		if m.fs.Exists(testPath) {
			patchPath = testPath
		}
	}

	if !m.fs.Exists(patchPath) {
		return fmt.Errorf("patch file not found: %s", patchFile)
	}

	// Check if patch is already applied
	checkArgs := []string{"-p1", "--dry-run", "-i", patchPath}
	err := m.exec.RunInDir(ctx, m.cfg.Paths.KernelDir, "patch", checkArgs...)
	if err != nil {
		// Try reverse check to see if already applied
		reverseArgs := []string{"-p1", "--dry-run", "-R", "-i", patchPath}
		if m.exec.RunInDir(ctx, m.cfg.Paths.KernelDir, "patch", reverseArgs...) == nil {
			return fmt.Errorf("patch appears to already be applied")
		}
		return fmt.Errorf("patch cannot be applied cleanly: %w", err)
	}

	// Apply the patch
	applyArgs := []string{"-p1", "-i", patchPath}
	if err := m.exec.RunInDir(ctx, m.cfg.Paths.KernelDir, "patch", applyArgs...); err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	return nil
}

// Reverse reverses a previously applied patch.
func (m *Manager) Reverse(ctx context.Context, patchFile string) error {
	patchPath := patchFile
	if !filepath.IsAbs(patchPath) {
		testPath := filepath.Join(m.cfg.Paths.PatchesDir, patchPath)
		if m.fs.Exists(testPath) {
			patchPath = testPath
		}
	}

	if !m.fs.Exists(patchPath) {
		return fmt.Errorf("patch file not found: %s", patchFile)
	}

	reverseArgs := []string{"-p1", "-R", "-i", patchPath}
	if err := m.exec.RunInDir(ctx, m.cfg.Paths.KernelDir, "patch", reverseArgs...); err != nil {
		return fmt.Errorf("failed to reverse patch: %w", err)
	}

	return nil
}

// List returns all available patches.
func (m *Manager) List() ([]PatchInfo, error) {
	if !m.fs.Exists(m.cfg.Paths.PatchesDir) {
		return nil, nil
	}

	var patches []PatchInfo

	// List version directories
	entries, err := m.fs.ReadDir(m.cfg.Paths.PatchesDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		versionDir := filepath.Join(m.cfg.Paths.PatchesDir, version)

		// List patch files in this version directory
		patchFiles, err := m.fs.ReadDir(versionDir)
		if err != nil {
			continue
		}

		for _, pf := range patchFiles {
			if pf.IsDir() {
				continue
			}

			name := pf.Name()
			if !strings.HasSuffix(name, ".patch") {
				continue
			}

			patches = append(patches, PatchInfo{
				Name:    name,
				Path:    filepath.Join(versionDir, name),
				Version: version,
			})
		}
	}

	return patches, nil
}

// GetPatchesForVersion returns patches for a specific kernel version.
func (m *Manager) GetPatchesForVersion(version string) ([]PatchInfo, error) {
	allPatches, err := m.List()
	if err != nil {
		return nil, err
	}

	var filtered []PatchInfo
	for _, p := range allPatches {
		if p.Version == version {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}
