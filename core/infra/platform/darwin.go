// Package platform provides OS-specific abstractions for ELMOS operations.
// This file implements the macOS platform using hdiutil and Homebrew.
package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/homebrew"
)

// darwinPlatform implements Platform for macOS.
type darwinPlatform struct {
	exec     executor.Executor
	diskImg  *darwinDiskImage
	packages *darwinPackages
	paths    *darwinPaths
}

func newDarwinPlatform(exec executor.Executor) *darwinPlatform {
	if exec == nil {
		exec = executor.NewShellExecutor()
	}
	resolver := homebrew.NewResolver(exec)
	return &darwinPlatform{
		exec:     exec,
		diskImg:  &darwinDiskImage{exec: exec},
		packages: &darwinPackages{exec: exec, resolver: resolver},
		paths:    &darwinPaths{exec: exec},
	}
}

func (d *darwinPlatform) Name() string                { return "darwin" }
func (d *darwinPlatform) DiskImage() DiskImageManager { return d.diskImg }
func (d *darwinPlatform) Packages() PackageManager    { return d.packages }
func (d *darwinPlatform) Paths() PathProvider         { return d.paths }

func (d *darwinPlatform) SetExecutor(exec executor.Executor) {
	d.exec = exec
	d.diskImg.exec = exec
	d.packages.exec = exec
	d.packages.resolver = homebrew.NewResolver(exec)
	d.paths.exec = exec
}

// --- DiskImageManager (macOS hdiutil) ---

type darwinDiskImage struct{ exec executor.Executor }

// Create allocates a sparse disk image using hdiutil.
// The imagePath should end in .sparseimage; hdiutil will produce a single file.
func (d *darwinDiskImage) Create(ctx context.Context, imagePath string, sizeGB int) error {
	// Skip if the image already exists
	if _, err := os.Stat(imagePath); err == nil {
		return nil
	}
	volname := strings.TrimSuffix(filepath.Base(imagePath), ".sparseimage")
	return d.exec.Run(ctx, "hdiutil", "create",
		"-size", fmt.Sprintf("%dg", sizeGB),
		"-fs", "Case-sensitive HFS+",
		"-type", "SPARSE",
		"-volname", volname,
		imagePath,
	)
}

// Mount attaches the image and returns the actual /Volumes/<name> mount point.
// It uses -mountpoint when the expected path is available; otherwise lets
// hdiutil choose and parses the output for the mount point.
func (d *darwinDiskImage) Mount(ctx context.Context, imagePath string) (string, error) {
	out, err := d.exec.Output(ctx, "hdiutil", "attach", "-nobrowse", imagePath)
	if err != nil {
		return "", fmt.Errorf("hdiutil attach %s: %w", imagePath, wrapExecError(err))
	}
	return parseDarwinMountPoint(out), nil
}

// Unmount detaches the image at mountPoint using hdiutil detach.
func (d *darwinDiskImage) Unmount(ctx context.Context, mountPoint string) error {
	return d.exec.Run(ctx, "hdiutil", "detach", mountPoint)
}

// IsMounted checks hdiutil info for an image identified by name.
func (d *darwinDiskImage) IsMounted(ctx context.Context, name string) (bool, string, error) {
	out, err := d.exec.Output(ctx, "hdiutil", "info")
	if err != nil {
		return false, "", fmt.Errorf("hdiutil info: %w", err)
	}
	mountPoint := parseDarwinMountedVolume(out, name)
	if mountPoint == "" {
		return false, "", nil
	}
	return true, mountPoint, nil
}

// parseDarwinMountPoint extracts the /Volumes/... path from hdiutil attach output.
func parseDarwinMountPoint(out []byte) string {
	for _, line := range bytes.Split(out, []byte("\n")) {
		if idx := bytes.Index(line, []byte("/Volumes/")); idx >= 0 {
			return strings.TrimSpace(string(line[idx:]))
		}
	}
	return ""
}

// parseDarwinMountedVolume scans hdiutil info output for a /Volumes/<name> entry.
// hdiutil info lines look like: "/dev/disk4s2  UUID  /Volumes/hello"
// It also handles the macOS convention where a duplicate volume gets a " 1" suffix.
func parseDarwinMountedVolume(out []byte, name string) string {
	target := "/Volumes/" + name
	for _, line := range bytes.Split(out, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		// Check if the line ends with the exact target path
		if strings.HasSuffix(trimmed, target) {
			return target
		}
		// Handle suffixed volumes like "/Volumes/hello 1"
		if idx := strings.Index(trimmed, target+" "); idx >= 0 {
			// Extract the actual mount point from end of line
			suffix := trimmed[idx:]
			return strings.TrimSpace(suffix)
		}
	}
	return ""
}

// wrapExecError extracts stderr from an exec.ExitError to provide better diagnostics.
func wrapExecError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return fmt.Errorf("%w (stderr: %s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return err
}

// --- PackageManager (Homebrew) ---

type darwinPackages struct {
	exec     executor.Executor
	resolver *homebrew.Resolver
}

func (d *darwinPackages) IsInstalled(pkg string) bool {
	return d.resolver.GetPrefix(pkg) != ""
}

func (d *darwinPackages) Install(ctx context.Context, pkg string) error {
	return d.exec.Run(ctx, "brew", "install", pkg)
}

func (d *darwinPackages) ListInstalled() ([]string, error) {
	out, err := d.exec.Output(context.Background(), "brew", "list", "--formula", "-1")
	if err != nil {
		return nil, fmt.Errorf("brew list: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result, nil
}

func (d *darwinPackages) GetBinPath(pkg string) string     { return d.resolver.GetBin(pkg) }
func (d *darwinPackages) GetLibPath(pkg string) string     { return d.resolver.GetLib(pkg) }
func (d *darwinPackages) GetIncludePath(pkg string) string { return d.resolver.GetInclude(pkg) }

// --- PathProvider (macOS) ---

type darwinPaths struct{ exec executor.Executor }

// WorkspaceRoot returns the /Volumes/<name> path used as the workspace mount point.
// TODO(platform): this is hardcoded macOS convention; ensure callers use this method.
func (d *darwinPaths) WorkspaceRoot(name string) string {
	return fmt.Sprintf("/Volumes/%s", name)
}

func (d *darwinPaths) CacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".elmos")
	}
	return filepath.Join(homeDir, ".elmos")
}

func (d *darwinPaths) ToolchainDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "x-tools")
	}
	return filepath.Join(homeDir, "x-tools")
}
