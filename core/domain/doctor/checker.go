// Package doctor provides dependency checking and environment validation for elmos.
package doctor

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/toolchain"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/platform"
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
	exec     executor.Executor
	fs       filesystem.FileSystem
	cfg      *elconfig.Config
	platform platform.Platform
	tm       *toolchain.Manager
}

// NewHealthChecker creates a new HealthChecker with the given dependencies.
func NewHealthChecker(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config, tm *toolchain.Manager) *HealthChecker {
	return &HealthChecker{
		exec:     exec,
		fs:       fs,
		cfg:      cfg,
		platform: platform.CurrentWithExecutor(exec),
		tm:       tm,
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

	// Check toolchains (ct-ng and installed targets)
	tcResults := h.CheckToolchains(ctx)
	results = append(results, tcResults...)

	return results, issues
}

// CheckToolchains checks if crosstool-ng and toolchains are installed.
func (h *HealthChecker) CheckToolchains(ctx context.Context) []CheckResult {
	var results []CheckResult

	// Check ct-ng installation
	ctngInstalled := h.tm.IsInstalled()
	results = append(results, CheckResult{
		Name:     "crosstool-ng",
		Passed:   ctngInstalled,
		Required: false, // Optional for building, but good to have
		Message:  "Run: elmos toolchains install",
	})

	if ctngInstalled {
		// Check installed toolchains
		toolchains, err := h.tm.GetInstalledToolchains()
		if err == nil {
			for _, tc := range toolchains {
				msg := ""
				if !tc.Installed {
					msg = "Toolchain built but not fully installed"
				}
				results = append(results, CheckResult{
					Name:     fmt.Sprintf("Toolchain: %s", tc.Target),
					Passed:   tc.Installed,
					Required: false,
					Message:  msg,
				})
			}
		}
	}

	return results
}

// CheckHomebrew checks the package manager for the current platform.
// On macOS it verifies Homebrew; on Linux it verifies apt-get/dnf/apk/pacman.
func (h *HealthChecker) CheckHomebrew(ctx context.Context) CheckResult {
	if runtime.GOOS == "darwin" {
		_, err := h.exec.LookPath("brew")
		return CheckResult{
			Name:     "Homebrew",
			Passed:   err == nil,
			Required: true,
			Message:  "Install from: https://brew.sh",
		}
	}
	// Linux: check for distro package manager
	mgr := h.detectLinuxPackageManager()
	return CheckResult{
		Name:     "Package Manager (" + h.platform.Name() + ")",
		Passed:   mgr != "",
		Required: true,
		Message:  "apt-get, dnf, apk, or pacman required",
	}
}

// detectLinuxPackageManager returns the name of the first found package manager.
func (h *HealthChecker) detectLinuxPackageManager() string {
	for _, mgr := range []string{"apt-get", "dnf", "apk", "pacman"} {
		if _, err := h.exec.LookPath(mgr); err == nil {
			return mgr
		}
	}
	return ""
}

// CheckPackages checks if required packages are installed.
func (h *HealthChecker) CheckPackages(ctx context.Context) []CheckResult {
	pkgSet, err := h.getInstalledPackageSet()
	if err != nil {
		return []CheckResult{{
			Name:     h.packageLabel(),
			Passed:   false,
			Required: true,
			Message:  "Failed to list packages",
		}}
	}

	grouped := h.groupPackagesByCategory()
	cats := []string{"Build Tools", "Virtualization", "Toolchain Dependencies"}

	var results []CheckResult
	for _, catName := range cats {
		results = append(results, h.checkCategoryPackages(catName, grouped[catName], pkgSet)...)
	}

	return results
}

// getInstalledPackageSet returns a set of installed package names.
// On Linux, native package names differ from canonical names (e.g. "libelf-dev"
// vs "libelf"), so we also probe each RequiredPackage by canonical name via
// platform.Packages().IsInstalled() to bridge the naming gap.
func (h *HealthChecker) getInstalledPackageSet() (map[string]bool, error) {
	installed, err := h.platform.Packages().ListInstalled()
	if err != nil {
		return nil, err
	}
	pkgSet := make(map[string]bool)
	for _, p := range installed {
		pkgSet[p] = true
	}
	// Also check each RequiredPackage by canonical name via platform IsInstalled.
	// This resolves canonical → native name internally and covers cases where
	// ListInstalled returns native names but RequiredPackages uses canonical ones.
	for _, pkg := range elconfig.RequiredPackages {
		if h.platform.Packages().IsInstalled(pkg.Name) {
			pkgSet[pkg.Name] = true
		}
	}
	return pkgSet, nil
}

// groupPackagesByCategory groups required packages by their category.
func (h *HealthChecker) groupPackagesByCategory() map[string][]elconfig.RequiredPackage {
	grouped := make(map[string][]elconfig.RequiredPackage)
	for _, pkg := range elconfig.RequiredPackages {
		cat := pkg.Category
		if cat == "" {
			cat = "Other"
		}
		grouped[cat] = append(grouped[cat], pkg)
	}
	return grouped
}

// checkCategoryPackages checks packages in a category and returns results.
func (h *HealthChecker) checkCategoryPackages(catName string, pkgs []elconfig.RequiredPackage, pkgSet map[string]bool) []CheckResult {
	if len(pkgs) == 0 {
		return nil
	}

	label := h.packageLabel()
	var results []CheckResult
	var missing []string

	for _, pkg := range pkgs {
		passed := pkgSet[pkg.Name]
		results = append(results, CheckResult{
			Name:     fmt.Sprintf("%s: [%s] %s", label, catName, pkg.Name),
			Passed:   passed,
			Required: pkg.Required,
			Message:  pkg.Description,
		})
		if !passed && pkg.Required {
			missing = append(missing, pkg.Name)
		}
	}

	if len(missing) > 0 {
		results = append(results, CheckResult{
			Name:     "  Fix missing packages",
			Passed:   false,
			Required: false,
			Message:  h.installHint(missing),
		})
	}

	return results
}

// packageLabel returns the section label for package check results.
func (h *HealthChecker) packageLabel() string {
	if runtime.GOOS == "darwin" {
		return "Homebrew Packages"
	}
	return "System Packages"
}

// installHint returns a platform-appropriate install command for the missing packages.
func (h *HealthChecker) installHint(missing []string) string {
	names := strings.Join(missing, " ")
	if runtime.GOOS == "darwin" {
		return "brew install " + names
	}
	platName := h.platform.Name()
	switch {
	case strings.Contains(platName, "debian"):
		return "sudo apt-get install " + names
	case strings.Contains(platName, "fedora"):
		return "sudo dnf install " + names
	case strings.Contains(platName, "alpine"):
		return "sudo apk add " + names
	case strings.Contains(platName, "arch"):
		return "sudo pacman -S " + names
	default:
		return "Install: " + names
	}
}

// CheckHeaders checks if required header files exist.
// On Linux, headers are at /usr/include (provided by libc-dev).
// On macOS, headers are in the project's assets/libraries/ directory.
func (h *HealthChecker) CheckHeaders(ctx context.Context) []CheckResult {
	if runtime.GOOS != "darwin" {
		return h.checkSystemHeaders()
	}
	return h.checkCustomHeaders()
}

// checkSystemHeaders checks for headers at /usr/include (Linux).
func (h *HealthChecker) checkSystemHeaders() []CheckResult {
	var results []CheckResult
	sysInclude := "/usr/include"
	for _, header := range elconfig.RequiredHeaders {
		passed := h.fs.Exists(filepath.Join(sysInclude, header))
		msg := ""
		if !passed {
			msg = "Install libc development headers (e.g., sudo apt-get install libc6-dev)"
		}
		results = append(results, CheckResult{
			Name:     fmt.Sprintf("System Headers: %s", header),
			Passed:   passed,
			Required: true,
			Message:  msg,
		})
	}
	return results
}

// checkCustomHeaders checks for headers in the project's assets/libraries/ (macOS).
func (h *HealthChecker) checkCustomHeaders() []CheckResult {
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
			Name:     fmt.Sprintf("Custom Headers: %s", header),
			Passed:   passed,
			Required: true,
			Message:  "",
		})
	}

	// Check asm directory
	asmDir := filepath.Join(headersDir, "asm")
	results = append(results, CheckResult{
		Name:     "Custom Headers: asm/",
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

		binary := archCfg.GDBBinary
		passed := false
		message := ""

		// Check elmos toolchains only (strict check)
		// Extract target tuple: binary minus "-gdb"
		if strings.HasSuffix(binary, "-gdb") {
			target := strings.TrimSuffix(binary, "-gdb")
			toolchainBin := filepath.Join(h.tm.Paths().XTools, target, "bin", binary)
			if h.fs.Exists(toolchainBin) {
				passed = true
				message = "Found in elmos toolchains"
			}
		}

		results = append(results, CheckResult{
			Name:     fmt.Sprintf("Cross Debuggers: %s (%s)", arch, binary),
			Passed:   passed,
			Required: false, // GDB is optional
			Message:  message,
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

		binary := archCfg.GCCBinary
		passed := false
		message := ""

		// Check elmos toolchains only (strict check)
		// Extract target tuple: binary minus "-gcc"
		if strings.HasSuffix(binary, "-gcc") {
			target := strings.TrimSuffix(binary, "-gcc")
			toolchainBin := filepath.Join(h.tm.Paths().XTools, target, "bin", binary)
			if h.fs.Exists(toolchainBin) {
				passed = true
				message = "Found in elmos toolchains"
			}
		}

		results = append(results, CheckResult{
			Name:     fmt.Sprintf("Cross Compilers: %s (%s)", arch, binary),
			Passed:   passed,
			Required: false, // GCC is optional when using LLVM
			Message:  message,
		})
	}

	return results
}

// IsElfHMissing checks if elf.h is missing.
func (h *HealthChecker) IsElfHMissing() bool {
	elfPath := filepath.Join(h.cfg.Paths.LibrariesDir, "elf.h")
	return !h.fs.Exists(elfPath)
}
