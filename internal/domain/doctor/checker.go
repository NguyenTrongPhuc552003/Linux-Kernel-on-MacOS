// Package doctor provides dependency checking and environment validation for elmos.
package doctor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/internal/config"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/homebrew"
)

// CheckResult represents the result of a single check.
type CheckResult struct {
	Name     string
	Passed   bool
	Required bool
	Message  string
}

// HealthChecker validates the development environment.
type HealthChecker struct {
	exec executor.Executor
	fs   filesystem.FileSystem
	cfg  *elconfig.Config
	brew *homebrew.Resolver
}

// NewHealthChecker creates a new HealthChecker with the given dependencies.
func NewHealthChecker(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config) *HealthChecker {
	return &HealthChecker{
		exec: exec,
		fs:   fs,
		cfg:  cfg,
		brew: homebrew.NewResolver(exec),
	}
}

// CheckAll runs all environment checks and returns the results.
func (h *HealthChecker) CheckAll(ctx context.Context) ([]CheckResult, int) {
	var results []CheckResult
	issues := 0

	// Check Homebrew
	result := h.CheckHomebrew(ctx)
	results = append(results, result)
	if !result.Passed && result.Required {
		issues++
	}

	// Check taps
	tapResults := h.CheckTaps(ctx)
	for _, r := range tapResults {
		results = append(results, r)
		if !r.Passed && r.Required {
			issues++
		}
	}

	// Check packages
	pkgResults := h.CheckPackages(ctx)
	for _, r := range pkgResults {
		results = append(results, r)
		if !r.Passed && r.Required {
			issues++
		}
	}

	// Check headers
	headerResults := h.CheckHeaders(ctx)
	for _, r := range headerResults {
		results = append(results, r)
		if !r.Passed && r.Required {
			issues++
		}
	}

	// Check cross debuggers
	gdbResults := h.CheckCrossGDB(ctx)
	results = append(results, gdbResults...)

	// Check cross compilers
	gccResults := h.CheckCrossGCC(ctx)
	results = append(results, gccResults...)

	return results, issues
}

// CheckHomebrew checks if Homebrew is installed.
func (h *HealthChecker) CheckHomebrew(ctx context.Context) CheckResult {
	_, err := h.exec.LookPath("brew")
	return CheckResult{
		Name:     "Homebrew",
		Passed:   err == nil,
		Required: true,
		Message:  "Install from: https://brew.sh",
	}
}

// CheckTaps checks if required Homebrew taps are installed.
func (h *HealthChecker) CheckTaps(ctx context.Context) []CheckResult {
	var results []CheckResult

	installedTaps, err := h.brew.ListTaps()
	if err != nil {
		return []CheckResult{{
			Name:     "Homebrew Taps",
			Passed:   false,
			Required: true,
			Message:  "Failed to list taps",
		}}
	}

	tapSet := make(map[string]bool)
	for _, t := range installedTaps {
		tapSet[t] = true
	}

	for _, tap := range elconfig.RequiredTaps {
		passed := tapSet[tap]
		results = append(results, CheckResult{
			Name:     fmt.Sprintf("Tap: %s", tap),
			Passed:   passed,
			Required: true,
			Message:  fmt.Sprintf("Fix: brew tap %s", tap),
		})
	}

	return results
}

// CheckPackages checks if required Homebrew packages are installed.
func (h *HealthChecker) CheckPackages(ctx context.Context) []CheckResult {
	var results []CheckResult

	installedPkgs, err := h.brew.ListInstalled()
	if err != nil {
		return []CheckResult{{
			Name:     "Homebrew Packages",
			Passed:   false,
			Required: true,
			Message:  "Failed to list packages",
		}}
	}

	pkgSet := make(map[string]bool)
	for _, p := range installedPkgs {
		pkgSet[p] = true
	}

	var missing []string
	for _, pkg := range elconfig.RequiredPackages {
		passed := pkgSet[pkg.Name]
		results = append(results, CheckResult{
			Name:     fmt.Sprintf("Package: %s", pkg.Name),
			Passed:   passed,
			Required: pkg.Required,
			Message:  pkg.Description,
		})
		if !passed && pkg.Required {
			missing = append(missing, pkg.Name)
		}
	}

	// Add fix message for missing packages
	if len(missing) > 0 {
		results = append(results, CheckResult{
			Name:     "Missing packages fix",
			Passed:   false,
			Required: false,
			Message:  fmt.Sprintf("brew install %s", strings.Join(missing, " ")),
		})
	}

	return results
}

// CheckHeaders checks if required header files exist.
func (h *HealthChecker) CheckHeaders(ctx context.Context) []CheckResult {
	var results []CheckResult

	headersDir := h.cfg.Paths.LibrariesDir

	if !h.fs.IsDir(headersDir) {
		return []CheckResult{{
			Name:     "Headers directory",
			Passed:   false,
			Required: true,
			Message:  fmt.Sprintf("Directory not found: %s", headersDir),
		}}
	}

	for _, header := range elconfig.RequiredHeaders {
		headerPath := filepath.Join(headersDir, header)
		passed := h.fs.Exists(headerPath)
		results = append(results, CheckResult{
			Name:     fmt.Sprintf("Header: %s", header),
			Passed:   passed,
			Required: true,
			Message:  "",
		})
	}

	// Check asm directory
	asmDir := filepath.Join(headersDir, "asm")
	results = append(results, CheckResult{
		Name:     "Header: asm/",
		Passed:   h.fs.IsDir(asmDir),
		Required: true,
		Message:  "",
	})

	return results
}

// CheckCrossGDB checks for cross-architecture GDB binaries.
func (h *HealthChecker) CheckCrossGDB(ctx context.Context) []CheckResult {
	var results []CheckResult

	for _, arch := range elconfig.SupportedArchitectures() {
		archCfg := elconfig.GetArchConfig(arch)
		if archCfg == nil || archCfg.GDBBinary == "" {
			continue
		}

		_, err := h.exec.LookPath(archCfg.GDBBinary)
		results = append(results, CheckResult{
			Name:     fmt.Sprintf("GDB: %s (%s)", arch, archCfg.GDBBinary),
			Passed:   err == nil,
			Required: false, // GDB is optional
			Message:  "",
		})
	}

	return results
}

// CheckCrossGCC checks for cross-architecture GCC binaries.
func (h *HealthChecker) CheckCrossGCC(ctx context.Context) []CheckResult {
	var results []CheckResult

	for _, arch := range elconfig.SupportedArchitectures() {
		archCfg := elconfig.GetArchConfig(arch)
		if archCfg == nil || archCfg.GCCBinary == "" {
			continue
		}

		_, err := h.exec.LookPath(archCfg.GCCBinary)
		results = append(results, CheckResult{
			Name:     fmt.Sprintf("GCC: %s (%s)", arch, archCfg.GCCBinary),
			Passed:   err == nil,
			Required: false, // GCC is optional when using LLVM
			Message:  "",
		})
	}

	return results
}

// IsElfHMissing checks if elf.h is missing.
func (h *HealthChecker) IsElfHMissing() bool {
	elfPath := filepath.Join(h.cfg.Paths.LibrariesDir, "elf.h")
	return !h.fs.Exists(elfPath)
}
