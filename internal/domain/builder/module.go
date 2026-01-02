// Package builder provides kernel and module build orchestration for elmos.
package builder

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/internal/config"
	elcontext "github.com/NguyenTrongPhuc552003/elmos/internal/context"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
)

// ModuleInfo contains information about a kernel module.
type ModuleInfo struct {
	Name        string
	Path        string
	Description string
	Built       bool
}

// ModuleBuilder orchestrates kernel module build operations.
type ModuleBuilder struct {
	exec executor.Executor
	fs   filesystem.FileSystem
	cfg  *elconfig.Config
	ctx  *elcontext.Context
}

// NewModuleBuilder creates a new ModuleBuilder with the given dependencies.
func NewModuleBuilder(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config, ctx *elcontext.Context) *ModuleBuilder {
	return &ModuleBuilder{
		exec: exec,
		fs:   fs,
		cfg:  cfg,
		ctx:  ctx,
	}
}

// Build builds one or all kernel modules.
func (m *ModuleBuilder) Build(ctx context.Context, name string) error {
	modules, err := m.GetModules(name)
	if err != nil {
		return err
	}

	if len(modules) == 0 {
		return nil // No modules to build
	}

	for _, mod := range modules {
		if err := m.buildModule(ctx, mod); err != nil {
			return err
		}
	}

	return nil
}

// buildModule builds a single module.
func (m *ModuleBuilder) buildModule(ctx context.Context, mod ModuleInfo) error {
	args := []string{
		"-C", m.cfg.Paths.KernelDir,
		fmt.Sprintf("M=%s", mod.Path),
		fmt.Sprintf("ARCH=%s", m.cfg.Build.Arch),
		"LLVM=1",
		fmt.Sprintf("CROSS_COMPILE=%s", m.cfg.Build.CrossCompile),
		"modules",
	}

	env := m.ctx.GetMakeEnv()
	return m.exec.RunWithEnv(ctx, env, "make", args...)
}

// Clean cleans one or all kernel modules.
func (m *ModuleBuilder) Clean(ctx context.Context, name string) error {
	modules, err := m.GetModules(name)
	if err != nil {
		return err
	}

	for _, mod := range modules {
		args := []string{
			"-C", m.cfg.Paths.KernelDir,
			fmt.Sprintf("M=%s", mod.Path),
			fmt.Sprintf("ARCH=%s", m.cfg.Build.Arch),
			"clean",
		}

		// Ignore errors during clean
		_ = m.exec.Run(ctx, "make", args...)
	}

	return nil
}

// GetModules returns a list of modules, optionally filtered by name.
func (m *ModuleBuilder) GetModules(name string) ([]ModuleInfo, error) {
	if name != "" {
		return m.getSpecificModule(name)
	}
	return m.getAllModules()
}

// getSpecificModule returns a single module by name.
func (m *ModuleBuilder) getSpecificModule(name string) ([]ModuleInfo, error) {
	modPath := filepath.Join(m.cfg.Paths.ModulesDir, name)
	if !m.fs.Exists(modPath) {
		return nil, fmt.Errorf("module not found: %s", name)
	}

	info := m.getModuleInfo(name, modPath)
	return []ModuleInfo{info}, nil
}

// getAllModules returns all modules in the modules directory.
func (m *ModuleBuilder) getAllModules() ([]ModuleInfo, error) {
	entries, err := m.fs.ReadDir(m.cfg.Paths.ModulesDir)
	if err != nil {
		return nil, err
	}

	var modules []ModuleInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		modPath := filepath.Join(m.cfg.Paths.ModulesDir, name)

		// Check for Makefile
		makePath := filepath.Join(modPath, "Makefile")
		if !m.fs.Exists(makePath) {
			continue
		}

		info := m.getModuleInfo(name, modPath)
		modules = append(modules, info)
	}

	return modules, nil
}

// getModuleInfo builds ModuleInfo for a module.
func (m *ModuleBuilder) getModuleInfo(name, path string) ModuleInfo {
	info := ModuleInfo{
		Name: name,
		Path: path,
	}

	// Check if built
	koFile := filepath.Join(path, name+".ko")
	info.Built = m.fs.Exists(koFile)

	// Extract description from source file
	srcFile := filepath.Join(path, name+".c")
	if content, err := m.fs.ReadFile(srcFile); err == nil {
		info.Description = extractModuleDescription(string(content))
	}

	return info
}

// extractModuleDescription extracts MODULE_DESCRIPTION from source code.
func extractModuleDescription(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "MODULE_DESCRIPTION") {
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start >= 0 && end > start {
				return line[start+1 : end]
			}
		}
	}
	return ""
}

// PrepareHeaders runs modules_prepare to set up headers for module building.
func (m *ModuleBuilder) PrepareHeaders(ctx context.Context) error {
	args := []string{
		"-C", m.cfg.Paths.KernelDir,
		fmt.Sprintf("-j%d", m.cfg.Build.Jobs),
		fmt.Sprintf("ARCH=%s", m.cfg.Build.Arch),
		"LLVM=1",
		fmt.Sprintf("CROSS_COMPILE=%s", m.cfg.Build.CrossCompile),
		"modules_prepare",
	}

	env := m.ctx.GetMakeEnv()
	return m.exec.RunWithEnv(ctx, env, "make", args...)
}

// CreateModule creates a new module from template.
func (m *ModuleBuilder) CreateModule(name string) error {
	modPath := filepath.Join(m.cfg.Paths.ModulesDir, name)

	// Check if already exists
	if m.fs.Exists(modPath) {
		return fmt.Errorf("module already exists: %s", name)
	}

	// Create directory
	if err := m.fs.MkdirAll(modPath, 0755); err != nil {
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
	if err := m.fs.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		return err
	}

	// Create Makefile
	makeContent := fmt.Sprintf(`obj-m += %s.o

# Optional: Add extra source files
# %s-objs := %s.o helper.o
`, name, name, name)

	makePath := filepath.Join(modPath, "Makefile")
	return m.fs.WriteFile(makePath, []byte(makeContent), 0644)
}
