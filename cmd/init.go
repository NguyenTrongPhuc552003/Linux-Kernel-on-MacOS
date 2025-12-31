// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// initCmd - initialize workspace (mount + clone)
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize workspace (mount image and clone kernel)",
	Long: `Initialize the ELMOS workspace by:
1. Creating/mounting the sparse disk image
2. Cloning the Linux kernel source if needed`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Mount image
		if err := runImageMount(); err != nil {
			return err
		}

		// Clone kernel if needed
		if err := runRepoCheck(); err != nil {
			return err
		}

		printSuccess("Workspace initialized successfully!")
		return nil
	},
}

// imageCmd - disk image management
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage sparse disk image",
	Long:  `Commands to create, mount, and unmount the case-sensitive APFS sparse image.`,
}

var imageMountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount the sparse image",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImageMount()
	},
}

var imageUnmountCmd = &cobra.Command{
	Use:     "unmount",
	Aliases: []string{"umount"},
	Short:   "Unmount the sparse image",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImageUnmount()
	},
}

var imageCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new sparse image",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImageCreate()
	},
}

var imageStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show image mount status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ctx.IsMounted() {
			printSuccess("Image is mounted at %s", ctx.Config.Image.MountPoint)
		} else {
			printWarn("Image is not mounted")
		}
		return nil
	},
}

func init() {
	imageCmd.AddCommand(imageMountCmd)
	imageCmd.AddCommand(imageUnmountCmd)
	imageCmd.AddCommand(imageCreateCmd)
	imageCmd.AddCommand(imageStatusCmd)
}

func runImageMount() error {
	cfg := ctx.Config

	// Check if already mounted
	if ctx.IsMounted() {
		printInfo("Volume already mounted at %s", cfg.Image.MountPoint)
		return nil
	}

	// Check if image exists, create if not
	if _, err := os.Stat(cfg.Image.Path); os.IsNotExist(err) {
		printStep("Creating %s sparse image...", cfg.Image.Size)
		if err := runImageCreate(); err != nil {
			return err
		}
	}

	// Mount the image
	printStep("Mounting %s...", cfg.Image.VolumeName)
	cmd := exec.Command("hdiutil", "attach", cfg.Image.Path, "-quiet")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount image: %w", err)
	}

	printSuccess("Mounted at %s", cfg.Image.MountPoint)
	return nil
}

func runImageUnmount() error {
	cfg := ctx.Config

	if !ctx.IsMounted() {
		printInfo("Volume is not mounted")
		return nil
	}

	printStep("Unmounting %s...", cfg.Image.MountPoint)
	cmd := exec.Command("hdiutil", "detach", cfg.Image.MountPoint, "-force")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unmount image: %w", err)
	}

	printSuccess("Unmounted successfully")
	return nil
}

func runImageCreate() error {
	cfg := ctx.Config

	// Check if already exists
	if _, err := os.Stat(cfg.Image.Path); err == nil {
		printWarn("Image already exists: %s", cfg.Image.Path)
		return nil
	}

	printStep("Creating %s case-sensitive APFS sparse image...", cfg.Image.Size)

	cmd := exec.Command("hdiutil", "create",
		"-size", cfg.Image.Size,
		"-fs", "Case-sensitive APFS",
		"-type", "SPARSE",
		"-volname", cfg.Image.VolumeName,
		cfg.Image.Path,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	printSuccess("Created image at %s", cfg.Image.Path)
	return nil
}
