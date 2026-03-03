// Package platform provides OS-specific abstractions for ELMOS operations.
// It hides platform differences (macOS, Linux, Windows) behind a uniform
// interface so the rest of the codebase remains OS-agnostic.
//
// Usage:
//
//	p := platform.Current()
//	if err := p.DiskImage().Create(imagePath, sizeGB); err != nil { ... }
//	mountPoint, err := p.DiskImage().Mount(imagePath)
//	workspace := p.Paths().WorkspaceRoot("myboard")
package platform

import (
	"context"

	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
)

// Platform is the root abstraction for all OS-specific operations.
// Obtain an instance with platform.Current() or platform.CurrentWithExecutor().
type Platform interface {
	// Name identifies the platform (e.g. "darwin", "linux-debian", "windows-wsl2").
	Name() string

	// DiskImage returns a manager for creating and mounting disk images.
	DiskImage() DiskImageManager

	// Packages returns a manager for querying and installing system packages.
	Packages() PackageManager

	// Paths returns OS-specific default path conventions.
	Paths() PathProvider
}

// DiskImageManager abstracts sparse disk image lifecycle management.
// macOS uses hdiutil; Linux uses fallocate + losetup; Windows delegates to WSL2.
type DiskImageManager interface {
	// Create allocates a new sparse image of sizeGB gigabytes at imagePath.
	Create(ctx context.Context, imagePath string, sizeGB int) error

	// Mount attaches the image and returns the mount point path.
	Mount(ctx context.Context, imagePath string) (mountPoint string, err error)

	// Unmount detaches the mounted image at mountPoint.
	Unmount(ctx context.Context, mountPoint string) error

	// IsMounted checks whether a workspace image (identified by name) is already
	// mounted. Returns (true, mountPoint, nil) on success.
	IsMounted(ctx context.Context, name string) (bool, string, error)
}

// PackageManager abstracts querying and installing system packages.
// macOS uses Homebrew; Linux uses apt/dnf/apk; Windows delegates to WSL2.
type PackageManager interface {
	// IsInstalled returns true if pkg appears to be installed.
	IsInstalled(pkg string) bool

	// Install installs pkg via the platform package manager.
	Install(ctx context.Context, pkg string) error

	// ListInstalled returns the names of all installed packages known to the manager.
	ListInstalled() ([]string, error)

	// GetBinPath returns the path to the bin directory of pkg.
	// Returns "" if the package is not found.
	GetBinPath(pkg string) string

	// GetLibPath returns the path to the lib directory of pkg.
	GetLibPath(pkg string) string

	// GetIncludePath returns the path to the include directory of pkg.
	GetIncludePath(pkg string) string
}

// PathProvider supplies OS-specific default path conventions.
type PathProvider interface {
	// WorkspaceRoot returns the canonical mount/root path for a named workspace.
	// macOS: /Volumes/<name>  Linux: /mnt/elmos/<name>  Windows: \\wsl$\Ubuntu\elmos\<name>
	WorkspaceRoot(name string) string

	// CacheDir returns the user-level ELMOS cache root (~/.elmos/).
	// Identical on all platforms.
	CacheDir() string

	// ToolchainDir returns the preferred directory for cross-compilation toolchains.
	// macOS/Linux: ~/x-tools/  Windows: ~/x-tools/ inside WSL2.
	ToolchainDir() string
}

// ExecutorAware is implemented by Platform impls that accept an executor.
// This lets the rest of the codebase inject its executor without knowing
// the concrete type.
type ExecutorAware interface {
	SetExecutor(exec executor.Executor)
}
