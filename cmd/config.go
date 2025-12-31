// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/NguyenTrongPhuc552003/elmos/internal/core"
)

// configCmd - configuration management
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage elmos configuration",
	Long:  `View and modify elmos configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunConfigShow()
	},
}

// RunConfigShow prints the current configuration
func RunConfigShow() error {
	cfg := ctx.Config
	fmt.Println("Current Configuration:")
	fmt.Println()
	fmt.Println("Image:")
	fmt.Printf("  Path:        %s\n", cfg.Image.Path)
	fmt.Printf("  Volume Name: %s\n", cfg.Image.VolumeName)
	fmt.Printf("  Size:        %s\n", cfg.Image.Size)
	fmt.Printf("  Mount Point: %s\n", cfg.Image.MountPoint)
	fmt.Println()
	fmt.Println("Build:")
	fmt.Printf("  Architecture:  %s\n", cfg.Build.Arch)
	fmt.Printf("  Jobs:          %d\n", cfg.Build.Jobs)
	fmt.Printf("  LLVM:          %t\n", cfg.Build.LLVM)
	fmt.Printf("  Cross Compile: %s\n", cfg.Build.CrossCompile)
	fmt.Println()
	fmt.Println("QEMU:")
	fmt.Printf("  Memory:   %s\n", cfg.QEMU.Memory)
	fmt.Printf("  GDB Port: %d\n", cfg.QEMU.GDBPort)
	fmt.Printf("  SMP:      %d\n", cfg.QEMU.SMP)
	fmt.Println()
	fmt.Println("Paths:")
	fmt.Printf("  Project Root:  %s\n", cfg.Paths.ProjectRoot)
	fmt.Printf("  Kernel Dir:    %s\n", cfg.Paths.KernelDir)
	fmt.Printf("  Modules Dir:   %s\n", cfg.Paths.ModulesDir)
	fmt.Printf("  Apps Dir:      %s\n", cfg.Paths.AppsDir)
	fmt.Printf("  Libraries Dir: %s\n", cfg.Paths.LibrariesDir)
	fmt.Printf("  Patches Dir:   %s\n", cfg.Paths.PatchesDir)
	return nil
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Available keys:
  arch          - Target architecture (arm64, riscv, arm)
  jobs          - Number of parallel build jobs
  memory        - QEMU memory size (e.g., 2G, 4G)
  volume_name   - Disk image volume name
  image_size    - Disk image size (e.g., 20G)`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		return runConfigSet(key, value)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigGet(args[0])
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new elmos.yaml configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigInit()
	},
}

var configProfileCmd = &cobra.Command{
	Use:   "profile [name]",
	Short: "Apply a named configuration profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigProfile(args[0])
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configProfileCmd)
}

func runConfigSet(key, value string) error {
	cfg := ctx.Config

	switch key {
	case "arch":
		if !isValidArch(value) {
			return fmt.Errorf("invalid architecture: %s (valid: arm64, riscv, arm)", value)
		}
		cfg.Build.Arch = value
	case "jobs":
		var jobs int
		if _, err := fmt.Sscanf(value, "%d", &jobs); err != nil || jobs < 1 {
			return fmt.Errorf("invalid jobs value: %s", value)
		}
		cfg.Build.Jobs = jobs
	case "memory":
		cfg.QEMU.Memory = value
	case "volume_name":
		cfg.Image.VolumeName = value
	case "image_size":
		cfg.Image.Size = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Save configuration
	configPath := filepath.Join(cfg.Paths.ProjectRoot, "elmos.yaml")
	if err := core.SaveConfig(cfg, configPath); err != nil {
		return err
	}

	printSuccess("Set %s = %s", key, value)
	return nil
}

func runConfigGet(key string) error {
	cfg := ctx.Config

	var value interface{}
	switch key {
	case "arch":
		value = cfg.Build.Arch
	case "jobs":
		value = cfg.Build.Jobs
	case "memory":
		value = cfg.QEMU.Memory
	case "volume_name":
		value = cfg.Image.VolumeName
	case "image_size":
		value = cfg.Image.Size
	case "kernel_dir":
		value = cfg.Paths.KernelDir
	case "modules_dir":
		value = cfg.Paths.ModulesDir
	case "apps_dir":
		value = cfg.Paths.AppsDir
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	fmt.Printf("%s = %v\n", key, value)
	return nil
}

func runConfigInit() error {
	cfg := ctx.Config
	configPath := filepath.Join(cfg.Paths.ProjectRoot, "elmos.yaml")

	// Check if already exists
	if _, err := os.Stat(configPath); err == nil {
		printWarn("Configuration file already exists: %s", configPath)
		return nil
	}

	// Create default configuration
	v := viper.New()
	v.SetConfigType("yaml")

	// Set values from current config
	v.Set("image.volume_name", cfg.Image.VolumeName)
	v.Set("image.size", cfg.Image.Size)
	v.Set("build.arch", cfg.Build.Arch)
	v.Set("build.jobs", cfg.Build.Jobs)
	v.Set("build.llvm", cfg.Build.LLVM)
	v.Set("build.cross_compile", cfg.Build.CrossCompile)
	v.Set("qemu.memory", cfg.QEMU.Memory)
	v.Set("qemu.gdb_port", cfg.QEMU.GDBPort)
	v.Set("paths.debian_mirror", cfg.Paths.DebianMirror)

	// Add example profiles
	v.Set("profiles.riscv-dev.arch", "riscv")
	v.Set("profiles.riscv-dev.memory", "2G")
	v.Set("profiles.arm64-dev.arch", "arm64")
	v.Set("profiles.arm64-dev.memory", "4G")

	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	printSuccess("Created configuration file: %s", configPath)
	return nil
}

func runConfigProfile(name string) error {
	if err := ctx.Config.ApplyProfile(name); err != nil {
		return err
	}

	printSuccess("Applied profile: %s", name)
	printInfo("Architecture: %s", ctx.Config.Build.Arch)
	return nil
}

func isValidArch(arch string) bool {
	valid := []string{"arm64", "riscv", "arm", "x86_64", "x86"}
	for _, a := range valid {
		if arch == a {
			return true
		}
	}
	return false
}
