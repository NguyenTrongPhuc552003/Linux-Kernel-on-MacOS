// Package toolchain provides crosstool-ng integration for building cross-compilers.
package toolchain

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/core/ui"
)

// Manager handles crosstool-ng toolchain operations.
type Manager struct {
	exec    executor.Executor
	fs      filesystem.FileSystem
	cfg     *elconfig.Config
	printer *ui.Printer
}

// NewManager creates a new toolchain Manager.
func NewManager(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config, printer *ui.Printer) *Manager {
	return &Manager{
		exec:    exec,
		fs:      fs,
		cfg:     cfg,
		printer: printer,
	}
}

// ToolchainInfo contains information about a built toolchain.
type ToolchainInfo struct {
	Target    string // e.g., "riscv64-unknown-linux-gnu"
	Path      string // Full path to toolchain directory
	Installed bool   // Whether fully built
	Version   string // GCC version if installed
}

// Paths returns important toolchain-related paths.
func (m *Manager) Paths() ToolchainPaths {
	base := m.cfg.Paths.ToolchainsDir
	return ToolchainPaths{
		Base:        base,
		CrosstoolNG: filepath.Join(base, "crosstool-ng"),
		XTools:      filepath.Join(base, "x-tools"),
		Src:         filepath.Join(base, "src"),
		Configs:     filepath.Join(base, "configs"),
	}
}

// ToolchainPaths holds all toolchain directory paths.
type ToolchainPaths struct {
	Base        string // /Volumes/elmos/toolchains
	CrosstoolNG string // ct-ng installation
	XTools      string // Built toolchains output
	Src         string // Downloaded tarballs cache
	Configs     string // Custom .config files
}

// IsInstalled checks if crosstool-ng is installed.
func (m *Manager) IsInstalled() bool {
	ctngBin := filepath.Join(m.Paths().CrosstoolNG, "bin", "ct-ng")
	return m.fs.Exists(ctngBin)
}

// GetCtNgPath returns the path to ct-ng binary.
func (m *Manager) GetCtNgPath() string {
	return filepath.Join(m.Paths().CrosstoolNG, "bin", "ct-ng")
}

// ListSamples lists available crosstool-ng sample configurations.
func (m *Manager) ListSamples(ctx context.Context) ([]string, error) {
	if !m.IsInstalled() {
		return nil, fmt.Errorf("crosstool-ng not installed, run 'elmos toolchains clone'")
	}

	// Use os/exec directly to stream output
	cmd := exec.CommandContext(ctx, m.GetCtNgPath(), "list-samples")

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Stream and parse output
	var samples []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse ct-ng output: "[L..]   riscv64-unknown-linux-gnu"
		// We look for lines starting with brackets
		trimLine := strings.TrimSpace(line)
		if len(trimLine) > 0 && strings.HasPrefix(trimLine, "[") {
			parts := strings.Fields(trimLine)
			if len(parts) >= 2 {
				sample := parts[len(parts)-1]
				samples = append(samples, sample)
			}
		}
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("failed to list samples: %w", err)
	}

	return samples, nil
}

// GetCustomConfigPath returns the path to a custom config file for the target.
// Returns empty string if no custom config exists.
func (m *Manager) GetCustomConfigPath(target string) string {
	// Check in project root assets/toolchains/configs
	projectConfigs := filepath.Join(m.cfg.Paths.ProjectRoot, "assets", "toolchains", "configs")
	configPath := filepath.Join(projectConfigs, target+".config")
	if m.fs.Exists(configPath) {
		return configPath
	}

	// Fallback to checking in the toolchains dir (if manually placed there)
	configPath = filepath.Join(m.Paths().Configs, target+".config")
	if m.fs.Exists(configPath) {
		return configPath
	}

	return ""
}

// GetInstalledToolchains returns a list of installed toolchains.
func (m *Manager) GetInstalledToolchains() ([]ToolchainInfo, error) {
	xtoolsDir := m.Paths().XTools
	if !m.fs.IsDir(xtoolsDir) {
		return nil, nil
	}

	entries, err := m.fs.ReadDir(xtoolsDir)
	if err != nil {
		return nil, err
	}

	var toolchains []ToolchainInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		target := entry.Name()
		path := filepath.Join(xtoolsDir, target)

		// Check if bin directory exists (indicates successful build)
		binDir := filepath.Join(path, "bin")
		installed := m.fs.IsDir(binDir)

		toolchains = append(toolchains, ToolchainInfo{
			Target:    target,
			Path:      path,
			Installed: installed,
		})
	}

	return toolchains, nil
}
