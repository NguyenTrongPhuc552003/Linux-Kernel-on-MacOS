// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/internal/core"
	"github.com/NguyenTrongPhuc552003/elmos/pkg/version"
)

var (
	// Global flags
	cfgFile     string
	verbose     bool
	interactive bool

	// Global context
	ctx *core.Context

	// Styles
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // Green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // Red
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))  // Yellow
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))  // Blue
	accentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))  // Purple
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "elmos",
	Short: "Embedded Linux on MacOS - Native kernel build tools",
	Long: `ELMOS (Embedded Linux on MacOS) provides native Linux kernel build tools 
for macOS, targeting RISC-V, ARM64, and more architectures.

No Docker, no VMs—just Clang/LLVM, Homebrew, and targeted patches 
for host tool compatibility.

Common workflow:
  elmos doctor              # Check dependencies
  elmos init                # Mount workspace and clone kernel
  elmos config set arch arm64  # Set target architecture
  elmos kernel config       # Configure kernel
  elmos build               # Build kernel
  elmos qemu run            # Test in QEMU`,
	Version: version.Get().String(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for certain commands
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		// Load configuration
		cfg, err := core.LoadConfig()
		if err != nil {
			return err
		}

		// Initialize global context
		ctx = core.NewContext(cfg)
		ctx.Verbose = verbose

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./elmos.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "enable interactive TUI mode")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(imageCmd)
	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(kernelCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(moduleCmd)
	rootCmd.AddCommand(qemuCmd)
	rootCmd.AddCommand(rootfsCmd)
	rootCmd.AddCommand(patchCmd)
}

// Helper functions for consistent output

func printSuccess(format string, args ...interface{}) {
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ "+format, args...)))
}

func printError(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, errorStyle.Render(fmt.Sprintf("✗ "+format, args...)))
}

func printWarn(format string, args ...interface{}) {
	fmt.Println(warnStyle.Render(fmt.Sprintf("⚠ "+format, args...)))
}

func printInfo(format string, args ...interface{}) {
	fmt.Println(infoStyle.Render(fmt.Sprintf("ℹ "+format, args...)))
}

func printStep(format string, args ...interface{}) {
	fmt.Println(accentStyle.Render(fmt.Sprintf("→ "+format, args...)))
}

// versionCmd - show version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		info := version.Get()
		fmt.Printf("ELMOS - Embedded Linux on MacOS\n")
		fmt.Printf("Version:    %s\n", accentStyle.Render(info.Version))
		fmt.Printf("Commit:     %s\n", info.Commit)
		fmt.Printf("Built:      %s\n", info.BuildDate)
		fmt.Printf("Go version: %s\n", info.GoVersion)
		fmt.Printf("OS/Arch:    %s/%s\n", info.OS, info.Arch)
	},
}
