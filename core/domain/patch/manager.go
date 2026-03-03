// Package patch provides kernel patch management for elmos.
package patch

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
)

var ErrPathTraversal = errors.New("patch: path escapes allowed directories")

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
	patchPath, err := m.resolvePatchPath(patchFile)
	if err != nil {
		return err
	}

	// Check if patch is already applied
	checkArgs := []string{"-p1", "--dry-run", "-i", patchPath}
	err = m.exec.RunInDir(ctx, m.cfg.Paths.KernelDir, "patch", checkArgs...)
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
	patchPath, err := m.resolvePatchPath(patchFile)
	if err != nil {
		return err
	}

	reverseArgs := []string{"-p1", "-R", "-i", patchPath}
	if err := m.exec.RunInDir(ctx, m.cfg.Paths.KernelDir, "patch", reverseArgs...); err != nil {
		return fmt.Errorf("failed to reverse patch: %w", err)
	}

	return nil
}

func (m *Manager) resolvePatchPath(patchFile string) (string, error) {
	patchPath := patchFile

	if !filepath.IsAbs(patchPath) {
		if strings.Contains(patchPath, "..") {
			return "", fmt.Errorf("invalid patch path %q: %w", patchFile, ErrPathTraversal)
		}
		patchPath = strings.TrimPrefix(patchPath, "patches/")
		patchPath = filepath.Join(m.cfg.Paths.PatchesDir, patchPath)
	}

	if err := m.validatePathConfinement(patchPath); err != nil {
		return "", err
	}

	if !m.fs.Exists(patchPath) {
		return "", fmt.Errorf("patch file not found: %s", patchFile)
	}

	return patchPath, nil
}

func (m *Manager) validatePathConfinement(targetPath string) error {
	cleanTarget, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err)
	}

	allowedBases := []string{m.cfg.Paths.PatchesDir, m.cfg.Paths.ProjectRoot}
	for _, base := range allowedBases {
		cleanBase, baseErr := filepath.Abs(filepath.Clean(base))
		if baseErr != nil {
			continue
		}
		if cleanTarget == cleanBase || strings.HasPrefix(cleanTarget, cleanBase+string(filepath.Separator)) {
			return nil
		}
	}

	return fmt.Errorf("path %q escapes workspace boundaries: %w", targetPath, ErrPathTraversal)
}

// List returns all available patches.
func (m *Manager) List() ([]PatchInfo, error) {
	if !m.fs.Exists(m.cfg.Paths.PatchesDir) {
		return nil, nil
	}

	versionDirs, err := m.fs.ReadDir(m.cfg.Paths.PatchesDir)
	if err != nil {
		return nil, err
	}

	var patches []PatchInfo
	for _, versionEntry := range versionDirs {
		if !versionEntry.IsDir() {
			continue
		}
		versionPatches := m.listPatchesForVersion(versionEntry.Name())
		patches = append(patches, versionPatches...)
	}

	return patches, nil
}

// listPatchesForVersion lists all patches for a specific version directory.
func (m *Manager) listPatchesForVersion(version string) []PatchInfo {
	versionDir := filepath.Join(m.cfg.Paths.PatchesDir, version)
	archDirs, err := m.fs.ReadDir(versionDir)
	if err != nil {
		return nil
	}

	var patches []PatchInfo
	for _, archEntry := range archDirs {
		if !archEntry.IsDir() {
			continue
		}
		archPatches := m.listPatchesForArch(version, archEntry.Name())
		patches = append(patches, archPatches...)
	}
	return patches
}

// listPatchesForArch lists all patches for a specific arch directory.
func (m *Manager) listPatchesForArch(version, arch string) []PatchInfo {
	archDir := filepath.Join(m.cfg.Paths.PatchesDir, version, arch)
	patchFiles, err := m.fs.ReadDir(archDir)
	if err != nil {
		return nil
	}

	var patches []PatchInfo
	for _, pf := range patchFiles {
		if pf.IsDir() || !strings.HasSuffix(pf.Name(), ".patch") {
			continue
		}
		patches = append(patches, PatchInfo{
			Name:    pf.Name(),
			Path:    filepath.Join(archDir, pf.Name()),
			Version: version,
			Arch:    arch,
		})
	}
	return patches
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
