// Package rootfs provides rootfs creation functionality for elmos.
package rootfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
)

// Creator handles rootfs creation.
type Creator struct {
	exec executor.Executor
	fs   filesystem.FileSystem
	cfg  *elconfig.Config
}

// NewCreator creates a new rootfs Creator.
func NewCreator(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config) *Creator {
	return &Creator{
		exec: exec,
		fs:   fs,
		cfg:  cfg,
	}
}

// Create creates an ext4 disk image with Debian rootfs.
func (c *Creator) Create(ctx context.Context, opts CreateOptions) error {
	size := opts.Size
	if size == "" {
		size = "5G"
	}

	diskImage := c.cfg.Paths.DiskImage
	rootfsDir := c.cfg.Paths.RootfsDir
	if err := c.validateManagedPath("rootfs", rootfsDir); err != nil {
		return err
	}
	if err := c.validateManagedPath("disk image", diskImage); err != nil {
		return err
	}

	if err := c.cleanRootfsDir(ctx, rootfsDir); err != nil {
		return err
	}

	if err := c.runDebootstrap(ctx, rootfsDir); err != nil {
		return err
	}

	if err := c.createInitScript(rootfsDir); err != nil {
		return fmt.Errorf("failed to create init script: %w", err)
	}

	if err := c.removeDiskImage(ctx, diskImage); err != nil {
		return err
	}

	if err := c.fixAptLists(ctx, rootfsDir); err != nil {
		fmt.Printf("Warning: failed to fix apt lists: %v\n", err)
	}

	return c.createDiskImage(ctx, diskImage, rootfsDir, size)
}

// cleanRootfsDir cleans old rootfs directory and creates fresh one.
func (c *Creator) cleanRootfsDir(ctx context.Context, rootfsDir string) error {
	if c.fs.Exists(rootfsDir) {
		if err := c.exec.Run(ctx, "sudo", "rm", "-rf", rootfsDir); err != nil {
			return fmt.Errorf("failed to clean old rootfs: %w", err)
		}
	}
	if err := c.exec.Run(ctx, "mkdir", "-p", rootfsDir); err != nil {
		return fmt.Errorf("failed to create rootfs directory: %w", err)
	}
	return nil
}

// runDebootstrap runs debootstrap to create the base Debian rootfs.
func (c *Creator) runDebootstrap(ctx context.Context, rootfsDir string) error {
	arch := c.getDebianArch()
	debootstrapDir := filepath.Join(c.cfg.Paths.ProjectRoot, "tools", "debootstrap")
	debootstrapPath := filepath.Join(debootstrapDir, "debootstrap")

	if !c.fs.Exists(debootstrapPath) {
		return fmt.Errorf("debootstrap not found at %s", debootstrapPath)
	}

	if err := c.exec.Run(ctx,
		"sudo", "-E", "DEBOOTSTRAP_DIR="+debootstrapDir,
		"fakeroot", debootstrapPath,
		"--foreign",
		"--arch="+arch,
		"--no-check-gpg",
		"stable",
		rootfsDir,
		c.cfg.Paths.DebianMirror,
	); err != nil {
		return fmt.Errorf("debootstrap failed: %w", err)
	}
	return nil
}

// removeDiskImage removes old disk image if exists.
func (c *Creator) removeDiskImage(ctx context.Context, diskImage string) error {
	if c.fs.Exists(diskImage) {
		if err := c.exec.Run(ctx, "rm", "-f", diskImage); err != nil {
			return fmt.Errorf("failed to remove old disk image: %w", err)
		}
	}
	return nil
}

// createDiskImage creates ext4 disk image from rootfs directory.
func (c *Creator) createDiskImage(ctx context.Context, diskImage, rootfsDir, size string) error {
	if err := c.exec.Run(ctx,
		"sudo", "mke2fs", "-t", "ext4",
		"-E", "lazy_itable_init=0,lazy_journal_init=0",
		"-d", rootfsDir,
		diskImage, size,
	); err != nil {
		return fmt.Errorf("failed to create disk image: %w", err)
	}
	return nil
}

// fixAptLists smoothes over differences between debootstrap versions by symlinking
// http:__ prefix files to their non-prefixed counterparts.
func (c *Creator) fixAptLists(ctx context.Context, rootfsDir string) error {
	listsDir := filepath.Join(rootfsDir, "var", "lib", "apt", "lists")
	entries, err := os.ReadDir(listsDir)
	if err != nil {
		return nil // Directory might not exist yet, ignore
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}

		newName := name
		if len(name) > 7 && name[:7] == "http:__" {
			newName = name[7:]
		} else if len(name) > 8 && name[:8] == "https:__" {
			newName = name[8:]
		}

		if newName != name {
			newPath := filepath.Join(listsDir, newName)
			if err := c.exec.Run(ctx, "sudo", "ln", "-sf", name, newPath); err != nil {
				return fmt.Errorf("failed to fix apt list entry %q: %w", name, err)
			}
		}
	}
	return nil
}

// getDebianArch returns the Debian architecture name.
func (c *Creator) getDebianArch() string {
	switch c.cfg.Build.Arch {
	case "arm64":
		return "arm64"
	case "arm":
		return "armhf"
	case "riscv":
		return "riscv64"
	default:
		return "arm64"
	}
}

// createInitScript creates the /init script for the rootfs.
func (c *Creator) createInitScript(rootfsDir string) error {
	initPath := filepath.Join(rootfsDir, "init")

	// Read init script from scripts/init file
	scriptPath := filepath.Join(c.cfg.Paths.ProjectRoot, "scripts", "init")
	content, err := c.fs.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read init script from %s: %w", scriptPath, err)
	}

	if err := c.fs.WriteFile(initPath, content, 0755); err != nil {
		return err
	}

	return nil
}

// RootfsInfo contains information about the rootfs.
type RootfsInfo struct {
	DiskImageExists bool
	DiskImagePath   string
	DiskImageSize   int64
	RootfsDirExists bool
	RootfsDirPath   string
	Architecture    string
}

// Status returns information about the current rootfs.
func (c *Creator) Status() (*RootfsInfo, error) {
	info := &RootfsInfo{
		DiskImagePath: c.cfg.Paths.DiskImage,
		RootfsDirPath: c.cfg.Paths.RootfsDir,
		Architecture:  c.getDebianArch(),
	}

	info.DiskImageExists = c.fs.Exists(c.cfg.Paths.DiskImage)
	info.RootfsDirExists = c.fs.Exists(c.cfg.Paths.RootfsDir)

	if info.DiskImageExists {
		if fi, err := os.Stat(c.cfg.Paths.DiskImage); err == nil {
			info.DiskImageSize = fi.Size()
		}
	}

	return info, nil
}

// Clean removes the rootfs directory and disk image.
func (c *Creator) Clean(ctx context.Context) error {
	if err := c.validateManagedPath("disk image", c.cfg.Paths.DiskImage); err != nil {
		return err
	}
	if err := c.validateManagedPath("rootfs", c.cfg.Paths.RootfsDir); err != nil {
		return err
	}

	// Remove disk image
	if c.fs.Exists(c.cfg.Paths.DiskImage) {
		if err := c.exec.Run(ctx, "rm", "-f", c.cfg.Paths.DiskImage); err != nil {
			return fmt.Errorf("failed to remove disk image: %w", err)
		}
	}

	// Remove rootfs directory (needs sudo due to root-owned files)
	if c.fs.Exists(c.cfg.Paths.RootfsDir) {
		if err := c.exec.Run(ctx, "sudo", "rm", "-rf", c.cfg.Paths.RootfsDir); err != nil {
			return fmt.Errorf("failed to remove rootfs directory: %w", err)
		}
	}

	return nil
}

func (c *Creator) validateManagedPath(kind, path string) error {
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to resolve %s path %q: %w", kind, path, err)
	}

	allowedRoots := []string{c.cfg.Paths.ProjectRoot, c.cfg.Image.MountPoint}
	for _, root := range allowedRoots {
		if root == "" {
			continue
		}
		cleanRoot, rootErr := filepath.Abs(filepath.Clean(root))
		if rootErr != nil {
			continue
		}
		if cleanPath == cleanRoot || strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) {
			return nil
		}
	}

	return fmt.Errorf("refusing unsafe %s path outside workspace boundaries: %s", kind, path)
}

// Exists returns true if the rootfs disk image exists.
func (c *Creator) Exists() bool {
	return c.fs.Exists(c.cfg.Paths.DiskImage)
}
