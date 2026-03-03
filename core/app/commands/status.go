package commands

import (
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// BuildStatus creates the status command for workspace status display.
func BuildStatus(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show workspace status (volume mount info)",
		RunE: func(cmd *cobra.Command, args []string) error {
			plat := ctx.AppContext.Platform
			name := ctx.Config.Image.VolumeName

			// Ask the platform layer for authoritative mount status.
			mounted, mountPoint, err := plat.DiskImage().IsMounted(cmd.Context(), name)
			if err != nil {
				ctx.Printer.Warn("Could not query mount status: %v", err)
				mounted = false
			}

			if !mounted {
				ctx.Printer.Info("Workspace not mounted")
				return nil
			}

			ctx.Printer.Success("Workspace mounted at %s", mountPoint)
			ctx.Printer.Print("")
			ctx.Printer.Step("Volume info:")

			// Platform-specific volume detail display.
			if runtime.GOOS == "darwin" {
				printDarwinVolumeInfo(ctx, cmd, mountPoint)
			} else {
				printLinuxVolumeInfo(ctx, cmd, mountPoint)
			}

			return nil
		},
	}
}

// printDarwinVolumeInfo displays hdiutil info filtered to our image (macOS).
func printDarwinVolumeInfo(ctx *Context, cmd *cobra.Command, mountPoint string) {
	out, err := ctx.Exec.Output(cmd.Context(), "hdiutil", "info")
	if err != nil {
		ctx.Printer.Warn("hdiutil info failed: %v", err)
		return
	}
	lines := strings.Split(string(out), "\n")
	inOurImage := false
	for _, line := range lines {
		if strings.Contains(line, ctx.Config.Image.Path) {
			inOurImage = true
		}
		if inOurImage {
			ctx.Printer.Print("  %s", line)
			if strings.HasPrefix(line, "/dev/disk") && strings.Contains(line, "/Volumes/") {
				break
			}
		}
	}
}

// printLinuxVolumeInfo displays loop device and filesystem info (Linux).
func printLinuxVolumeInfo(ctx *Context, cmd *cobra.Command, mountPoint string) {
	printMountEntry(ctx, cmd, mountPoint)
	printDiskUsage(ctx, cmd, mountPoint)
	printSparseInfo(ctx, cmd)
	printLoopDevice(ctx, cmd)
}

func printMountEntry(ctx *Context, cmd *cobra.Command, mountPoint string) {
	data, err := ctx.Exec.Output(cmd.Context(), "grep", mountPoint, "/proc/mounts")
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			ctx.Printer.Print("  %s", line)
		}
	}
}

func printDiskUsage(ctx *Context, cmd *cobra.Command, mountPoint string) {
	data, err := ctx.Exec.Output(cmd.Context(), "df", "-h", mountPoint)
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		ctx.Printer.Print("  %s", line)
	}
}

func printSparseInfo(ctx *Context, cmd *cobra.Command) {
	imgPath := ctx.Config.Image.Path
	if imgPath == "" {
		return
	}
	if data, err := ctx.Exec.Output(cmd.Context(), "du", "-h", imgPath); err == nil {
		ctx.Printer.Print("  Sparse file: %s", strings.TrimSpace(string(data)))
	}
	if data, err := ctx.Exec.Output(cmd.Context(), "ls", "-lh", imgPath); err == nil {
		fields := strings.Fields(strings.TrimSpace(string(data)))
		if len(fields) >= 5 {
			ctx.Printer.Print("  Logical size: %s", fields[4])
		}
	}
}

func printLoopDevice(ctx *Context, cmd *cobra.Command) {
	data, err := ctx.Exec.Output(cmd.Context(), "sudo", "losetup", "-a")
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.Contains(line, ctx.Config.Image.VolumeName) {
			ctx.Printer.Print("  %s", line)
		}
	}
}
