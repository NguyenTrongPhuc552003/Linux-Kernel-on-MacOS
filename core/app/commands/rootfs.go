package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/core/domain/rootfs"
)

// BuildRootfs creates the rootfs command tree for root filesystem management.
func BuildRootfs(ctx *Context) *cobra.Command {
	rootfsCmd := &cobra.Command{
		Use:   "rootfs",
		Short: "Manage root filesystem",
	}

	// Create command
	var size string
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build rootfs with Debian",
		Long:  "Build an ext4 disk image with Debian rootfs using debootstrap",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ctx.AppContext.EnsureMounted(); err != nil {
				return err
			}
			ctx.Printer.Step("Building rootfs...")
			if err := ctx.RootfsCreator.Create(cmd.Context(), rootfs.CreateOptions{Size: size}); err != nil {
				return err
			}
			ctx.Printer.Success("Rootfs built!")
			return nil
		},
	}
	buildCmd.Flags().StringVarP(&size, "size", "s", "5G", "Disk image size (e.g., 5G, 10G)")

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show rootfs status",
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := ctx.RootfsCreator.Status()
			if err != nil {
				return err
			}

			ctx.Printer.Info("Rootfs Status")
			ctx.Printer.Print("")

			if info.DiskImageExists {
				sizeStr := formatBytes(info.DiskImageSize)
				ctx.Printer.Print("  Disk Image:   ✓ exists (%s)", sizeStr)
				ctx.Printer.Print("  Path:         %s", info.DiskImagePath)
			} else {
				ctx.Printer.Print("  Disk Image:   ✗ not created")
				ctx.Printer.Print("  Path:         %s", info.DiskImagePath)
			}

			ctx.Printer.Print("  Architecture: %s", info.Architecture)

			if info.RootfsDirExists {
				ctx.Printer.Print("  Rootfs Dir:   ✓ exists")
			} else {
				ctx.Printer.Print("  Rootfs Dir:   ✗ not created")
			}

			return nil
		},
	}

	// Clean command
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove rootfs and disk image",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx.Printer.Step("Cleaning rootfs...")
			if err := ctx.RootfsCreator.Clean(cmd.Context()); err != nil {
				return err
			}
			ctx.Printer.Success("Rootfs cleaned!")
			return nil
		},
	}

	rootfsCmd.AddCommand(buildCmd, statusCmd, cleanCmd)
	return rootfsCmd
}

// formatBytes formats bytes into human readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
