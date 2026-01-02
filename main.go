// Package main is the entry point for the elmos CLI application.
// ELMOS (Embedded Linux on MacOS) provides native Linux kernel build tools for macOS.
package main

import (
	"os"

	"github.com/NguyenTrongPhuc552003/elmos/internal/app"
	"github.com/NguyenTrongPhuc552003/elmos/internal/config"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
)

func main() {
	// Initialize infrastructure
	exec := executor.NewShellExecutor()
	fs := filesystem.NewOSFileSystem()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Continue with defaults on error
		cfg = config.Get()
	}

	// Create application with all dependencies wired
	application := app.New(exec, fs, cfg)

	// Build and execute root command
	rootCmd := application.BuildRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
