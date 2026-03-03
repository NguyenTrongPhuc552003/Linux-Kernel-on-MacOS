// Package platform provides OS-specific abstractions for ELMOS operations.
// This file implements the Windows platform by delegating all operations to
// a WSL2 (Windows Subsystem for Linux) environment via wsl.exe.
//
// Requirements:
//   - WSL2 installed with a Debian/Ubuntu distribution
//   - wsl.exe available on the system PATH
package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
)

// windowsPlatform implements Platform for Windows hosts via WSL2 delegation.
type windowsPlatform struct {
	exec     executor.Executor
	diskImg  *windowsDiskImage
	packages *windowsPackages
	paths    *windowsPaths
}

func newWindowsPlatform(exec executor.Executor) *windowsPlatform {
	if exec == nil {
		exec = executor.NewShellExecutor()
	}
	return &windowsPlatform{
		exec:     exec,
		diskImg:  &windowsDiskImage{exec: exec},
		packages: &windowsPackages{exec: exec},
		paths:    &windowsPaths{},
	}
}

func (w *windowsPlatform) Name() string                { return "windows-wsl2" }
func (w *windowsPlatform) DiskImage() DiskImageManager { return w.diskImg }
func (w *windowsPlatform) Packages() PackageManager    { return w.packages }
func (w *windowsPlatform) Paths() PathProvider         { return w.paths }

func (w *windowsPlatform) SetExecutor(exec executor.Executor) {
	w.exec = exec
	w.diskImg.exec = exec
	w.packages.exec = exec
}

// --- DiskImageManager (Windows: delegates to WSL2) ---

type windowsDiskImage struct{ exec executor.Executor }

// Create creates a sparse image inside the WSL2 filesystem using fallocate.
func (d *windowsDiskImage) Create(ctx context.Context, imagePath string, sizeGB int) error {
	wslPath := windowsToWSLPath(imagePath)
	sizeBytes := fmt.Sprintf("%d", int64(sizeGB)*1024*1024*1024)
	return d.exec.Run(ctx, "wsl.exe", "--", "fallocate", "-l", sizeBytes, wslPath)
}

// Mount mounts the image inside WSL2 at /mnt/elmos/<name>.
func (d *windowsDiskImage) Mount(ctx context.Context, imagePath string) (string, error) {
	wslPath := windowsToWSLPath(imagePath)
	name := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
	mountPoint := fmt.Sprintf("/mnt/elmos/%s", name)

	// Ensure mount directory exists
	if err := d.exec.Run(ctx, "wsl.exe", "--", "mkdir", "-p", mountPoint); err != nil {
		return "", fmt.Errorf("wsl mkdir: %w", err)
	}

	// Allocate loop device
	loopDevice, err := d.exec.Output(ctx, "wsl.exe", "--", "losetup", "--find", "--show", wslPath)
	if err != nil {
		return "", fmt.Errorf("wsl losetup: %w", err)
	}
	loopDev := strings.TrimSpace(string(loopDevice))

	// Mount the loop device
	if err := d.exec.Run(ctx, "wsl.exe", "--", "mount", loopDev, mountPoint); err != nil {
		// Cleanup: detach loop device on mount failure
		_ = d.exec.Run(ctx, "wsl.exe", "--", "losetup", "-d", loopDev)
		return "", fmt.Errorf("wsl mount: %w", err)
	}

	return mountPoint, nil
}

// Unmount unmounts inside WSL2.
func (d *windowsDiskImage) Unmount(ctx context.Context, mountPoint string) error {
	// Unmount first
	if err := d.exec.Run(ctx, "wsl.exe", "--", "umount", mountPoint); err != nil {
		return fmt.Errorf("wsl umount: %w", err)
	}

	// Find and detach loop device
	loopInfo, err := d.exec.Output(ctx, "wsl.exe", "--", "losetup", "-j", mountPoint)
	if err == nil && len(loopInfo) > 0 {
		// Extract loop device name (format: "/dev/loop0: ...")
		parts := strings.SplitN(string(loopInfo), ":", 2)
		if len(parts) > 0 {
			loopDev := strings.TrimSpace(parts[0])
			_ = d.exec.Run(ctx, "wsl.exe", "--", "losetup", "-d", loopDev)
		}
	}

	return nil
}

// IsMounted checks /proc/mounts inside WSL2.
func (d *windowsDiskImage) IsMounted(ctx context.Context, name string) (bool, string, error) {
	target := fmt.Sprintf("/mnt/elmos/%s", name)

	// Use grep to check /proc/mounts directly
	err := d.exec.Run(ctx, "wsl.exe", "--", "grep", "-q", " "+target+" ", "/proc/mounts")
	if err != nil {
		// grep returns non-zero exit code if not found
		return false, "", nil
	}

	return true, target, nil
}

// --- PackageManager (Windows: apt-get inside WSL2) ---

type windowsPackages struct{ exec executor.Executor }

func (p *windowsPackages) IsInstalled(pkg string) bool {
	out, err := p.exec.Output(context.Background(), "wsl.exe", "--", "dpkg", "-s", pkg)
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Status: install ok installed")
}

func (p *windowsPackages) Install(ctx context.Context, pkg string) error {
	return p.exec.Run(ctx, "wsl.exe", "--", "apt-get", "install", "-y", pkg)
}

func (p *windowsPackages) ListInstalled() ([]string, error) {
	out, err := p.exec.Output(context.Background(), "wsl.exe", "--", "dpkg", "--get-selections")
	if err != nil {
		return nil, fmt.Errorf("wsl dpkg --get-selections: %w", err)
	}
	var result []string
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			result = append(result, strings.Fields(line)[0])
		}
	}
	return result, nil
}

func (p *windowsPackages) GetBinPath(pkg string) string {
	out, err := p.exec.Output(context.Background(), "wsl.exe", "--", "which", pkg)
	if err != nil || len(out) == 0 {
		return ""
	}
	return filepath.Dir(strings.TrimSpace(string(out)))
}

func (p *windowsPackages) GetLibPath(_ string) string     { return "/usr/lib" }
func (p *windowsPackages) GetIncludePath(_ string) string { return "/usr/include" }

// --- PathProvider (Windows) ---

type windowsPaths struct{}

// WorkspaceRoot returns the UNC path to the workspace directory in WSL2.
// This allows Windows tools to access the workspace if needed.
func (p *windowsPaths) WorkspaceRoot(name string) string {
	return fmt.Sprintf("\\\\wsl$\\Ubuntu\\mnt\\elmos\\%s", name)
}

func (p *windowsPaths) CacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".elmos")
	}
	return filepath.Join(homeDir, ".elmos")
}

func (p *windowsPaths) ToolchainDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "x-tools")
	}
	return filepath.Join(homeDir, "x-tools")
}

// windowsToWSLPath converts a Windows-style path to a WSL2 /mnt/... path.
// e.g. C:\Users\foo\elmos.img → /mnt/c/Users/foo/elmos.img
func windowsToWSLPath(winPath string) string {
	if len(winPath) >= 2 && winPath[1] == ':' {
		drive := strings.ToLower(string(winPath[0]))
		rest := strings.ReplaceAll(winPath[2:], "\\", "/")
		return fmt.Sprintf("/mnt/%s%s", drive, rest)
	}
	return strings.ReplaceAll(winPath, "\\", "/")
}
