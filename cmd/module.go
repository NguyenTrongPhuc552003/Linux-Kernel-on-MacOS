// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// moduleCmd - kernel module management
var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Manage kernel modules",
	Long: `Build, load, and manage out-of-tree kernel modules.

Modules are stored in the modules/ directory and can be built
against the configured kernel, then loaded in QEMU via 9p share.`,
}

var moduleBuildCmd = &cobra.Command{
	Use:   "build [name]",
	Short: "Build kernel modules",
	Long:  `Build one or all kernel modules. If no name specified, builds all.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ctx.EnsureMounted(); err != nil {
			return err
		}
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		return runModuleBuild(name)
	},
}

var moduleCleanCmd = &cobra.Command{
	Use:   "clean [name]",
	Short: "Clean module build artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ctx.EnsureMounted(); err != nil {
			return err
		}
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		return runModuleClean(name)
	},
}

var moduleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show module build status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleStatus()
	},
}

var moduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available modules",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleList()
	},
}

var moduleNewCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new module from template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleNew(args[0])
	},
}

var moduleHeadersCmd = &cobra.Command{
	Use:   "headers",
	Short: "Prepare kernel headers for module building",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ctx.EnsureMounted(); err != nil {
			return err
		}
		return runModuleHeaders()
	},
}

func init() {
	moduleCmd.AddCommand(moduleBuildCmd)
	moduleCmd.AddCommand(moduleCleanCmd)
	moduleCmd.AddCommand(moduleStatusCmd)
	moduleCmd.AddCommand(moduleListCmd)
	moduleCmd.AddCommand(moduleNewCmd)
	moduleCmd.AddCommand(moduleHeadersCmd)
}

func runModuleBuild(name string) error {
	cfg := ctx.Config

	// Get list of modules to build
	modules, err := getModules(name)
	if err != nil {
		return err
	}

	if len(modules) == 0 {
		printInfo("No modules found to build")
		return nil
	}

	for _, modName := range modules {
		modPath := filepath.Join(cfg.Paths.ModulesDir, modName)

		printStep("Building module: %s", modName)

		cmd := exec.Command("make",
			"-C", cfg.Paths.KernelDir,
			fmt.Sprintf("M=%s", modPath),
			fmt.Sprintf("ARCH=%s", cfg.Build.Arch),
			"LLVM=1",
			fmt.Sprintf("CROSS_COMPILE=%s", cfg.Build.CrossCompile),
			"modules",
		)
		cmd.Env = ctx.GetMakeEnv()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			printError("Failed to build module: %s", modName)
			return err
		}

		printSuccess("Built: %s", modName)
	}

	return nil
}

func runModuleClean(name string) error {
	cfg := ctx.Config

	modules, err := getModules(name)
	if err != nil {
		return err
	}

	for _, modName := range modules {
		modPath := filepath.Join(cfg.Paths.ModulesDir, modName)

		printStep("Cleaning module: %s", modName)

		cmd := exec.Command("make",
			"-C", cfg.Paths.KernelDir,
			fmt.Sprintf("M=%s", modPath),
			fmt.Sprintf("ARCH=%s", cfg.Build.Arch),
			"clean",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			printWarn("Failed to clean module: %s", modName)
		}
	}

	printSuccess("Modules cleaned")
	return nil
}

func runModuleStatus() error {
	cfg := ctx.Config

	modules, _ := getModules("")
	if len(modules) == 0 {
		printInfo("No modules found")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-20s %-12s\n", "MODULE", "STATUS")
	fmt.Println("  " + strings.Repeat("-", 35))

	for _, modName := range modules {
		modPath := filepath.Join(cfg.Paths.ModulesDir, modName)
		koFile := filepath.Join(modPath, modName+".ko")

		status := "not built"
		if _, err := os.Stat(koFile); err == nil {
			status = successStyle.Render("✓ built")
		}

		fmt.Printf("  %-20s %s\n", modName, status)
	}

	fmt.Println()
	return nil
}

func runModuleList() error {
	cfg := ctx.Config

	modules, _ := getModules("")
	if len(modules) == 0 {
		printInfo("No modules found in %s", cfg.Paths.ModulesDir)
		return nil
	}

	fmt.Println("Available modules:")
	for i, mod := range modules {
		modPath := filepath.Join(cfg.Paths.ModulesDir, mod)
		srcFile := filepath.Join(modPath, mod+".c")

		desc := ""
		if content, err := os.ReadFile(srcFile); err == nil {
			// Extract MODULE_DESCRIPTION
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(line, "MODULE_DESCRIPTION") {
					start := strings.Index(line, "\"")
					end := strings.LastIndex(line, "\"")
					if start >= 0 && end > start {
						desc = line[start+1 : end]
					}
					break
				}
			}
		}

		fmt.Printf("  %d. %s", i+1, mod)
		if desc != "" {
			fmt.Printf(" - %s", desc)
		}
		fmt.Println()
	}

	return nil
}

func runModuleNew(name string) error {
	cfg := ctx.Config

	modPath := filepath.Join(cfg.Paths.ModulesDir, name)

	// Check if already exists
	if _, err := os.Stat(modPath); err == nil {
		return fmt.Errorf("module already exists: %s", name)
	}

	// Create directory
	if err := os.MkdirAll(modPath, 0755); err != nil {
		return err
	}

	// Create source file
	srcContent := fmt.Sprintf(`// SPDX-License-Identifier: GPL-2.0
/*
 * %s - Kernel module
 */

#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>

static int __init %s_init(void)
{
    pr_info("%s: Module loaded\n");
    return 0;
}

static void __exit %s_exit(void)
{
    pr_info("%s: Module unloaded\n");
}

module_init(%s_init);
module_exit(%s_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Your Name");
MODULE_DESCRIPTION("A simple kernel module");
MODULE_VERSION("1.0");
`, name, name, name, name, name, name, name)

	srcPath := filepath.Join(modPath, name+".c")
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		return err
	}

	// Create Makefile
	makeContent := fmt.Sprintf(`obj-m += %s.o

# Optional: Add extra source files
# %s-objs := %s.o helper.o
`, name, name, name)

	makePath := filepath.Join(modPath, "Makefile")
	if err := os.WriteFile(makePath, []byte(makeContent), 0644); err != nil {
		return err
	}

	printSuccess("Created module: %s", modPath)
	printInfo("Edit %s/%s.c to implement your module", modPath, name)
	return nil
}

func runModuleHeaders() error {
	printStep("Preparing kernel headers for module building...")

	// Run modules_prepare
	jobs := ctx.Config.Build.Jobs
	return runBuild(jobs, []string{"modules_prepare"})
}

func getModules(name string) ([]string, error) {
	cfg := ctx.Config

	if name != "" {
		// Check specific module exists
		modPath := filepath.Join(cfg.Paths.ModulesDir, name)
		if _, err := os.Stat(modPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("module not found: %s", name)
		}
		return []string{name}, nil
	}

	// Get all modules
	entries, err := os.ReadDir(cfg.Paths.ModulesDir)
	if err != nil {
		return nil, err
	}

	var modules []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check for Makefile
		makePath := filepath.Join(cfg.Paths.ModulesDir, name, "Makefile")
		if _, err := os.Stat(makePath); err == nil {
			modules = append(modules, name)
		}
	}

	return modules, nil
}
