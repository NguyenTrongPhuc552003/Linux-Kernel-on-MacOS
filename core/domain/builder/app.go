// Package builder provides kernel and module build orchestration for elmos.
package builder

import (
	"context"
	"fmt"
	"path/filepath"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
)

// AppInfo contains information about a userspace application.
type AppInfo struct {
	Name  string
	Path  string
	Built bool
}

// AppBuilder orchestrates userspace application build operations.
type AppBuilder struct {
	exec executor.Executor
	fs   filesystem.FileSystem
	cfg  *elconfig.Config
}

// NewAppBuilder creates a new AppBuilder with the given dependencies.
func NewAppBuilder(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config) *AppBuilder {
	return &AppBuilder{
		exec: exec,
		fs:   fs,
		cfg:  cfg,
	}
}

// Build builds one or all userspace applications.
func (a *AppBuilder) Build(ctx context.Context, name string) error {
	apps, err := a.GetApps(name)
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		return nil
	}

	compiler := a.getCrossCompiler()

	for _, app := range apps {
		if err := a.buildApp(ctx, app, compiler); err != nil {
			return err
		}
	}

	return nil
}

// buildApp builds a single application.
func (a *AppBuilder) buildApp(ctx context.Context, app AppInfo, compiler string) error {
	// Check for Makefile
	makefilePath := filepath.Join(app.Path, "Makefile")
	if a.fs.Exists(makefilePath) {
		return a.exec.RunInDir(ctx, app.Path, "make",
			fmt.Sprintf("CC=%s", compiler),
			fmt.Sprintf("ARCH=%s", a.cfg.Build.Arch),
		)
	}

	// Simple compilation
	srcFile := filepath.Join(app.Path, app.Name+".c")
	if !a.fs.Exists(srcFile) {
		return fmt.Errorf("no source file found for %s", app.Name)
	}

	outFile := filepath.Join(app.Path, app.Name)
	return a.exec.Run(ctx, compiler, "-static", "-o", outFile, srcFile)
}

// Clean cleans one or all applications.
func (a *AppBuilder) Clean(ctx context.Context, name string) error {
	apps, err := a.GetApps(name)
	if err != nil {
		return err
	}

	for _, app := range apps {
		makefilePath := filepath.Join(app.Path, "Makefile")
		if a.fs.Exists(makefilePath) {
			_ = a.exec.RunInDir(ctx, app.Path, "make", "clean")
		} else {
			binPath := filepath.Join(app.Path, app.Name)
			_ = a.fs.Remove(binPath)
		}
	}

	return nil
}

// GetApps returns a list of applications, optionally filtered by name.
func (a *AppBuilder) GetApps(name string) ([]AppInfo, error) {
	if !a.fs.Exists(a.cfg.Paths.AppsDir) {
		return nil, nil
	}

	if name != "" {
		return a.getSpecificApp(name)
	}
	return a.getAllApps()
}

// getSpecificApp returns a single app by name.
func (a *AppBuilder) getSpecificApp(name string) ([]AppInfo, error) {
	appPath := filepath.Join(a.cfg.Paths.AppsDir, name)
	if !a.fs.Exists(appPath) {
		return nil, fmt.Errorf("app not found: %s", name)
	}

	info := a.getAppInfo(name, appPath)
	return []AppInfo{info}, nil
}

// getAllApps returns all apps in the apps directory.
func (a *AppBuilder) getAllApps() ([]AppInfo, error) {
	entries, err := a.fs.ReadDir(a.cfg.Paths.AppsDir)
	if err != nil {
		return nil, err
	}

	var apps []AppInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		appPath := filepath.Join(a.cfg.Paths.AppsDir, name)

		// Check for source file or Makefile
		srcPath := filepath.Join(appPath, name+".c")
		makePath := filepath.Join(appPath, "Makefile")

		if !a.fs.Exists(srcPath) && !a.fs.Exists(makePath) {
			continue
		}

		info := a.getAppInfo(name, appPath)
		apps = append(apps, info)
	}

	return apps, nil
}

// getAppInfo builds AppInfo for an application.
func (a *AppBuilder) getAppInfo(name, path string) AppInfo {
	info := AppInfo{
		Name: name,
		Path: path,
	}

	// Check if built
	binPath := filepath.Join(path, name)
	info.Built = a.fs.Exists(binPath)

	return info
}

// getCrossCompiler returns the cross-compiler for the current architecture.
func (a *AppBuilder) getCrossCompiler() string {
	archCfg := a.cfg.GetArchConfig()
	if archCfg != nil && archCfg.GCCBinary != "" {
		if path, err := a.exec.LookPath(archCfg.GCCBinary); err == nil {
			return path
		}
	}
	return "clang"
}

// CreateApp creates a new application from template.
func (a *AppBuilder) CreateApp(name string) error {
	appPath := filepath.Join(a.cfg.Paths.AppsDir, name)

	// Ensure apps directory exists
	if err := a.fs.MkdirAll(a.cfg.Paths.AppsDir, 0755); err != nil {
		return err
	}

	// Check if already exists
	if a.fs.Exists(appPath) {
		return fmt.Errorf("app already exists: %s", name)
	}

	// Create directory
	if err := a.fs.MkdirAll(appPath, 0755); err != nil {
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
	if err := a.fs.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
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
	return a.fs.WriteFile(makePath, []byte(makeContent), 0644)
}
