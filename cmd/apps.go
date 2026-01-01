// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// appsCmd - userspace applications management
var appsCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage userspace applications",
	Long: `Build and manage userspace applications for the target architecture.

Apps are stored in the apps/ directory and cross-compiled for the target.`,
}

var appsBuildCmd = &cobra.Command{
	Use:   "build [name]",
	Short: "Build userspace applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		return runAppsBuild(name)
	},
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppsList()
	},
}

var appsNewCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new application from template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAppsNew(args[0])
	},
}

var appsCleanCmd = &cobra.Command{
	Use:   "clean [name]",
	Short: "Clean built application binaries",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		return runAppsClean(name)
	},
}

func init() {
	appsCmd.AddCommand(appsBuildCmd)
	appsCmd.AddCommand(appsListCmd)
	appsCmd.AddCommand(appsNewCmd)
	appsCmd.AddCommand(appsCleanCmd)

	// Add apps to root command (initially defined in root.go but adding here for safety)
	rootCmd.AddCommand(appsCmd)
}

func runAppsBuild(name string) error {
	cfg := ctx.Config

	apps, err := getApps(name)
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		printInfo("No apps found to build")
		return nil
	}

	// Get cross-compiler based on arch
	compiler := getCrossCompiler(cfg.Build.Arch)

	for _, appName := range apps {
		appPath := filepath.Join(cfg.Paths.AppsDir, appName)

		printStep("Building app: %s for %s", appName, cfg.Build.Arch)

		// Look for Makefile or simple C file
		makefilePath := filepath.Join(appPath, "Makefile")
		if _, err := os.Stat(makefilePath); err == nil {
			// Use Makefile
			cmd := exec.Command("make",
				"-C", appPath,
				fmt.Sprintf("CC=%s", compiler),
				fmt.Sprintf("ARCH=%s", cfg.Build.Arch),
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				printError("Failed to build app: %s", appName)
				return err
			}
		} else {
			// Simple compilation
			srcFile := filepath.Join(appPath, appName+".c")
			outFile := filepath.Join(appPath, appName)

			if _, err := os.Stat(srcFile); os.IsNotExist(err) {
				printWarn("No source file found for %s", appName)
				continue
			}

			cmd := exec.Command(compiler,
				"-static",
				"-o", outFile,
				srcFile,
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				printError("Failed to compile: %s", srcFile)
				return err
			}
		}

		printSuccess("Built: %s", appName)
	}

	return nil
}

func runAppsList() error {
	cfg := ctx.Config

	apps, _ := getApps("")
	if len(apps) == 0 {
		printInfo("No apps found in %s", cfg.Paths.AppsDir)
		printInfo("Create one with: elmos app new <name>")
		return nil
	}

	fmt.Println("Available applications:")
	for i, app := range apps {
		appPath := filepath.Join(cfg.Paths.AppsDir, app)

		// Check if built
		binPath := filepath.Join(appPath, app)
		status := ""
		if _, err := os.Stat(binPath); err == nil {
			status = " (built)"
		}

		fmt.Printf("  %d. %s%s\n", i+1, app, status)
	}

	return nil
}

func runAppsClean(name string) error {
	cfg := ctx.Config

	apps, err := getApps(name)
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		printInfo("No apps found to clean")
		return nil
	}

	for _, appName := range apps {
		appPath := filepath.Join(cfg.Paths.AppsDir, appName)

		printStep("Cleaning app: %s", appName)

		// Check for Makefile with clean target
		makefilePath := filepath.Join(appPath, "Makefile")
		if _, err := os.Stat(makefilePath); err == nil {
			cmd := exec.Command("make", "-C", appPath, "clean")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				printWarn("make clean failed for %s", appName)
			}
		} else {
			// Just remove the binary
			binPath := filepath.Join(appPath, appName)
			if err := os.Remove(binPath); err == nil {
				printSuccess("Removed: %s", binPath)
			}
		}
	}

	printSuccess("Clean complete")
	return nil
}

func runAppsNew(name string) error {
	cfg := ctx.Config

	appPath := filepath.Join(cfg.Paths.AppsDir, name)

	// Create apps directory if it doesn't exist
	if err := os.MkdirAll(cfg.Paths.AppsDir, 0755); err != nil {
		return err
	}

	// Check if already exists
	if _, err := os.Stat(appPath); err == nil {
		return fmt.Errorf("app already exists: %s", name)
	}

	// Create directory
	if err := os.MkdirAll(appPath, 0755); err != nil {
		return err
	}

	// Create source file
	srcContent := fmt.Sprintf(`/*
 * %s - Userspace application
 * Cross-compiled for embedded Linux
 */

#include <stdio.h>
#include <stdlib.h>

int main(int argc, char *argv[])
{
    printf("%s: Hello from embedded Linux!\n");
    return 0;
}
`, name, name)

	srcPath := filepath.Join(appPath, name+".c")
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		return err
	}

	// Create Makefile
	makeContent := fmt.Sprintf(`# %s Makefile
CC ?= clang
CFLAGS ?= -Wall -Wextra -static

%s: %s.c
	$(CC) $(CFLAGS) -o $@ $<

clean:
	rm -f %s

.PHONY: clean
`, name, name, name, name)

	makePath := filepath.Join(appPath, "Makefile")
	if err := os.WriteFile(makePath, []byte(makeContent), 0644); err != nil {
		return err
	}

	printSuccess("Created app: %s", appPath)
	printInfo("Edit %s/%s.c to implement your application", appPath, name)
	return nil
}

func getApps(name string) ([]string, error) {
	cfg := ctx.Config

	// Check if apps directory exists
	if _, err := os.Stat(cfg.Paths.AppsDir); os.IsNotExist(err) {
		return nil, nil // No apps directory
	}

	if name != "" {
		appPath := filepath.Join(cfg.Paths.AppsDir, name)
		if _, err := os.Stat(appPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("app not found: %s", name)
		}
		return []string{name}, nil
	}

	// Get all apps
	entries, err := os.ReadDir(cfg.Paths.AppsDir)
	if err != nil {
		return nil, err
	}

	var apps []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check for source file or Makefile
		srcPath := filepath.Join(cfg.Paths.AppsDir, name, name+".c")
		makePath := filepath.Join(cfg.Paths.AppsDir, name, "Makefile")
		if _, err := os.Stat(srcPath); err == nil {
			apps = append(apps, name)
		} else if _, err := os.Stat(makePath); err == nil {
			apps = append(apps, name)
		}
	}

	return apps, nil
}

func getCrossCompiler(arch string) string {
	// Toolchains from messense/macos-cross-toolchains tap
	gccCompilers := map[string]string{
		"arm64": "aarch64-unknown-linux-gnu-gcc",
		"riscv": "riscv64-unknown-linux-gnu-gcc",
		"arm":   "arm-unknown-linux-gnueabihf-gcc",
	}

	// Check for GCC cross-compiler first (has sysroot with libc)
	if gcc, ok := gccCompilers[arch]; ok {
		if path, err := exec.LookPath(gcc); err == nil {
			return path
		}
	}

	// Fallback: use host clang (only works for native arch)
	printWarn("Cross-compiler for %s not found", arch)
	printInfo("Install with: brew install %s", getToolchainPackage(arch))

	return "clang"
}

func getToolchainPackage(arch string) string {
	packages := map[string]string{
		"arm64": "messense/macos-cross-toolchains/aarch64-unknown-linux-gnu",
		"riscv": "messense/macos-cross-toolchains/riscv64-unknown-linux-gnu",
		"arm":   "messense/macos-cross-toolchains/arm-unknown-linux-gnueabihf",
	}
	if pkg, ok := packages[arch]; ok {
		return pkg
	}
	return "unknown"
}
