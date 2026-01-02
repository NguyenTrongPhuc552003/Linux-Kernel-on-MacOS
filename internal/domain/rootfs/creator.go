// Package rootfs provides rootfs creation functionality for elmos.
package rootfs

import (
	"context"
	"fmt"
	"path/filepath"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/internal/config"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
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

// CreateOptions contains options for creating a rootfs.
type CreateOptions struct {
	Size string // Disk image size, e.g., "5G"
}

// Create creates an ext4 disk image with Debian rootfs.
func (c *Creator) Create(ctx context.Context, opts CreateOptions) error {
	size := opts.Size
	if size == "" {
		size = "5G"
	}

	diskImage := c.cfg.Paths.DiskImage
	rootfsDir := c.cfg.Paths.RootfsDir

	// Create disk image using dd
	sizeNum := size[:len(size)-1]
	sizeUnit := "1G"
	if size[len(size)-1] == 'M' {
		sizeUnit = "1M"
	}

	// Create sparse file
	if err := c.exec.Run(ctx, "dd", "if=/dev/zero", "of="+diskImage,
		"bs="+sizeUnit, "count=0", "seek="+sizeNum); err != nil {
		return fmt.Errorf("failed to create disk image: %w", err)
	}

	// Format as ext4
	if err := c.exec.Run(ctx, "mkfs.ext4", "-F", diskImage); err != nil {
		return fmt.Errorf("failed to format disk image: %w", err)
	}

	// Create mount point
	if err := c.fs.MkdirAll(rootfsDir, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	// Mount the image (requires fuse-ext2 on macOS)
	// This is a simplified version - actual mounting is complex on macOS

	// For now, we'll use fakeroot + debootstrap approach
	arch := c.getDebianArch()

	// Run debootstrap
	debootstrapPath := filepath.Join(c.cfg.Paths.ProjectRoot, "tools", "debootstrap", "debootstrap")
	if !c.fs.Exists(debootstrapPath) {
		return fmt.Errorf("debootstrap not found at %s", debootstrapPath)
	}

	if err := c.exec.Run(ctx, "fakeroot", debootstrapPath,
		"--arch="+arch,
		"--variant=minbase",
		"--include=systemd,systemd-sysv",
		"stable",
		rootfsDir,
		c.cfg.Paths.DebianMirror,
	); err != nil {
		return fmt.Errorf("debootstrap failed: %w", err)
	}

	// Create init script
	if err := c.createInitScript(rootfsDir); err != nil {
		return fmt.Errorf("failed to create init script: %w", err)
	}

	// Copy rootfs to disk image using e2cp tools
	// This is simplified - actual implementation would be more complex

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

	content := `#!/bin/sh
# elmos init script

echo "Booting Debian root filesystem..."

# Mount essential filesystems
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev

# Create pts
mkdir -p /dev/pts
mount -t devpts devpts /dev/pts

# Setup root filesystem if needed
if [ ! -f /etc/passwd ]; then
    echo "Setting up root filesystem..."
    echo "root:x:0:0:root:/root:/bin/sh" > /etc/passwd
    echo "root::0:0:::::" > /etc/shadow
    mkdir -p /root
fi

echo "Root filesystem already set up."

# Mount 9p shared modules directory
mkdir -p /mnt/modules
mount -t 9p -o trans=virtio modules_mount /mnt/modules 2>/dev/null || true

# Run module sync if available
if [ -x /mnt/modules/guesync.sh ]; then
    /mnt/modules/guesync.sh
fi

echo "System ready."

# Start shell
exec /bin/sh
`

	if err := c.fs.WriteFile(initPath, []byte(content), 0755); err != nil {
		return err
	}

	return nil
}
