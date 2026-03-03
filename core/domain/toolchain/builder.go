// Package toolchain provides crosstool-ng integration for building cross-compilers.
package toolchain

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Install installs crosstool-ng from the latest git source.
func (m *Manager) Install(ctx context.Context) error {
	paths := m.Paths()

	// Ensure base directory exists
	if err := m.fs.MkdirAll(paths.Base, 0755); err != nil {
		return fmt.Errorf("failed to create toolchains directory: %w", err)
	}

	// Create src directory for downloads
	if err := m.fs.MkdirAll(paths.Src, 0755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	srcDir := filepath.Join(paths.Base, "crosstool-ng-src")

	// Clone crosstool-ng if not exists
	if !m.fs.IsDir(srcDir) {
		err := m.exec.Run(ctx, "git", "clone",
			"https://github.com/crosstool-ng/crosstool-ng.git",
			srcDir)
		if err != nil {
			return fmt.Errorf("failed to clone crosstool-ng: %w", err)
		}
	}

	// Get install environment with brew paths
	env := m.getInstallEnv()

	// Bootstrap
	if err := m.exec.RunWithEnvInDir(ctx, env, srcDir, "./bootstrap"); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Configure with prefix
	configArgs := []string{
		fmt.Sprintf("--prefix=%s", paths.CrosstoolNG),
	}
	if err := m.exec.RunWithEnvInDir(ctx, env, srcDir, "./configure", configArgs...); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}

	// Build
	if err := m.exec.RunWithEnvInDir(ctx, env, srcDir, "make"); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	// Install
	if err := m.exec.RunWithEnvInDir(ctx, env, srcDir, "make", "install"); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}

// getInstallEnv returns environment variables for installing ct-ng.
// On macOS, adds Homebrew binutils and bison to PATH.
// On Linux, these tools are already on the standard PATH.
func (m *Manager) getInstallEnv() []string {
	env := os.Environ()
	if runtime.GOOS != "darwin" {
		return env
	}

	brewPrefix := getBrewPrefix()
	currentPath := os.Getenv("PATH")
	newPath := fmt.Sprintf("%s/opt/binutils/bin:%s/opt/bison/bin:%s",
		brewPrefix, brewPrefix, currentPath)

	for i, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			env[i] = "PATH=" + newPath
			return env
		}
	}
	env = append(env, "PATH="+newPath)
	return env
}

// SelectTarget configures crosstool-ng for the specified target.
// It checks for a custom config first, then falls back to ct-ng samples.
func (m *Manager) SelectTarget(ctx context.Context, target string) error {
	m.printer.Step("Configuring toolchain for %s...", target)

	if !m.IsInstalled() {
		return fmt.Errorf("crosstool-ng not installed, run 'elmos toolchains install'")
	}

	paths := m.Paths()

	// Create x-tools directory with proper permissions
	if err := m.fs.MkdirAll(paths.XTools, 0755); err != nil {
		return fmt.Errorf("failed to create x-tools directory: %w", err)
	}

	// Check if we have a custom config for this target
	customConfig := m.GetCustomConfigPath(target)
	if customConfig != "" {
		m.printer.Info("Using custom configuration: %s", customConfig)

		// Read custom config
		content, err := os.ReadFile(customConfig)
		if err != nil {
			return fmt.Errorf("failed to read custom config: %w", err)
		}

		// Write to .config in toolchains dir
		configPath := filepath.Join(paths.Base, ".config")
		if err := os.WriteFile(configPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write .config: %w", err)
		}
	} else {
		// Use ct-ng sample
		// Run ct-ng <target> in toolchains directory
		if err := m.exec.RunWithEnvInDir(ctx, m.getBuildEnv(paths), paths.Base, m.GetCtNgPath(), target); err != nil {
			return fmt.Errorf("failed to select target %s: %w", target, err)
		}
	}

	// Update .config to use our paths
	configFile := filepath.Join(paths.Base, ".config")
	if m.fs.Exists(configFile) {
		if err := m.patchConfig(configFile, paths); err != nil {
			return fmt.Errorf("failed to patch config: %w", err)
		}
	}

	return nil
}

// Build builds the currently configured toolchain.
func (m *Manager) Build(ctx context.Context, jobs int) error {
	if !m.IsInstalled() {
		return fmt.Errorf("crosstool-ng not installed, run 'elmos toolchains install'")
	}

	paths := m.Paths()

	// Check for .config
	configFile := filepath.Join(paths.Base, ".config")
	if !m.fs.Exists(configFile) {
		return fmt.Errorf("no target selected, run 'elmos toolchains <target>' first")
	}

	// Setup environment
	env := m.getBuildEnv(paths)

	// Build with specified jobs
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}

	buildTarget := fmt.Sprintf("build.%d", jobs)
	if err := m.exec.RunWithEnvInDir(ctx, env, paths.Base, m.GetCtNgPath(), buildTarget); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}

// Clean cleans the build artifacts.
func (m *Manager) Clean(ctx context.Context) error {
	if !m.IsInstalled() {
		return nil
	}

	paths := m.Paths()
	return m.exec.RunInDir(ctx, paths.Base, m.GetCtNgPath(), "clean")
}

// Menuconfig opens the interactive configuration menu.
func (m *Manager) Menuconfig(ctx context.Context) error {
	if !m.IsInstalled() {
		return fmt.Errorf("crosstool-ng not installed, run 'elmos toolchains install'")
	}

	paths := m.Paths()
	return m.exec.RunInDir(ctx, paths.Base, m.GetCtNgPath(), "menuconfig")
}

// patchConfig updates the .config file to use our paths.
func (m *Manager) patchConfig(configFile string, paths ToolchainPaths) error {
	content, err := m.fs.ReadFile(configFile)
	if err != nil {
		return err
	}

	patched := patchConfigContent(string(content), paths)
	return m.fs.WriteFile(configFile, []byte(patched), 0644)
}

// patchConfigContent replaces home directory paths with our paths.
func patchConfigContent(content string, paths ToolchainPaths) string {
	home := os.Getenv("HOME")

	// Replace prefix directory
	content = replaceConfigValue(content, "CT_PREFIX_DIR",
		fmt.Sprintf("%s/${CT_TARGET}", paths.XTools))

	// Replace source directory
	content = replaceConfigValue(content, "CT_LOCAL_TARBALLS_DIR", paths.Src)

	// Replace home paths with our base
	if home != "" {
		content = replaceAll(content, home+"/x-tools", paths.XTools)
		content = replaceAll(content, home+"/src", paths.Src)
	}

	// On macOS only: disable companion tools that fail with Clang/GCC mixing.
	// On Linux, ct-ng builds these natively without issues.
	if runtime.GOOS == "darwin" {
		content = replaceAll(content, "CT_COMP_TOOLS_M4=y", "# CT_COMP_TOOLS_M4 is not set")
		content = replaceAll(content, "CT_COMP_TOOLS_MAKE=y", "# CT_COMP_TOOLS_MAKE is not set")
		content = replaceAll(content, "CT_COMP_TOOLS_AUTOCONF=y", "# CT_COMP_TOOLS_AUTOCONF is not set")
		content = replaceAll(content, "CT_COMP_TOOLS_AUTOMAKE=y", "# CT_COMP_TOOLS_AUTOMAKE is not set")
		content = replaceAll(content, "CT_COMP_TOOLS_LIBTOOL=y", "# CT_COMP_TOOLS_LIBTOOL is not set")
	}

	return content
}

// replaceConfigValue replaces a config value in the content.
func replaceConfigValue(content, key, value string) string {
	// Find and replace CT_KEY="..." pattern
	prefix := key + "=\""
	start := 0
	for {
		idx := indexOf(content[start:], prefix)
		if idx < 0 {
			break
		}
		idx += start
		endIdx := indexOf(content[idx+len(prefix):], "\"")
		if endIdx < 0 {
			break
		}
		endIdx += idx + len(prefix)
		content = content[:idx] + key + "=\"" + value + "\"" + content[endIdx+1:]
		start = idx + len(key) + len(value) + 3
	}
	return content
}

// replaceAll replaces all occurrences of old with new.
func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx < 0 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// getBuildEnv returns environment variables for building toolchains.
// Implements brianredbeard's solution from github.com/crosstool-ng/crosstool-ng/issues/2378.
func (m *Manager) getBuildEnv(paths ToolchainPaths) []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("CT_PREFIX=%s", paths.XTools))

	// Export workspace name for crosstool-NG config variable substitution
	// This allows toolchain configs to use ${ELMOS_WORKSPACE} in paths
	if m.cfg != nil && m.cfg.Image.VolumeName != "" {
		env = append(env, fmt.Sprintf("ELMOS_WORKSPACE=%s", m.cfg.Image.VolumeName))
	}

	// On Linux, make is GNU make by default, libraries are in standard paths,
	// and GCC is already on PATH. No special env manipulation needed.
	if runtime.GOOS != "darwin" {
		return env
	}

	// macOS: adjust paths for Homebrew libraries and GNU tools.
	brewPrefix := getBrewPrefix()
	localBin := m.ensureLocalBin()
	m.ensureGCCSymlinks(localBin, brewPrefix)

	env = m.addLibraryFlags(env, brewPrefix)
	env = m.addGmake(env, brewPrefix)
	env = m.buildPathEnv(env, localBin, brewPrefix)

	return env
}

// getBrewPrefix returns the Homebrew prefix.
// TODO(platform): replace with platform.Current().Packages().GetBinPath("brew") in Phase 4.
func getBrewPrefix() string {
	if prefix := os.Getenv("HOMEBREW_PREFIX"); prefix != "" {
		return prefix
	}
	return "/opt/homebrew" // TODO(platform): default only correct on Apple Silicon macOS
}

// ensureLocalBin creates and returns the ~/.local/bin directory.
func (m *Manager) ensureLocalBin() string {
	home := os.Getenv("HOME")
	localBin := filepath.Join(home, ".local", "bin")
	_ = m.fs.MkdirAll(localBin, 0755)
	return localBin
}

// ensureGCCSymlinks creates gcc/g++ symlinks in localBin if needed.
func (m *Manager) ensureGCCSymlinks(localBin, brewPrefix string) {
	gccLink := filepath.Join(localBin, "gcc")
	if m.fs.Exists(gccLink) {
		return
	}
	for _, ver := range []string{"14", "15", "13", "12"} {
		gccPath := fmt.Sprintf("%s/bin/gcc-%s", brewPrefix, ver)
		if m.fs.Exists(gccPath) {
			_ = os.Symlink(gccPath, gccLink)
			gxxPath := fmt.Sprintf("%s/bin/g++-%s", brewPrefix, ver)
			gxxLink := filepath.Join(localBin, "g++")
			if !m.fs.Exists(gxxLink) {
				_ = os.Symlink(gxxPath, gxxLink)
			}
			break
		}
	}
}

// addLibraryFlags adds LDFLAGS and CPPFLAGS for Homebrew libraries.
func (m *Manager) addLibraryFlags(env []string, brewPrefix string) []string {
	ldPkgs := []string{"bison", "flex", "ncurses", "zlib"}
	cppPkgs := []string{"binutils", "flex", "ncurses", "zlib"}

	ldflags := m.buildFlagString(brewPrefix, ldPkgs, "lib", "-L")
	if ldflags != "" {
		env = appendOrUpdateEnv(env, "LDFLAGS", ldflags)
	}

	cppflags := m.buildFlagString(brewPrefix, cppPkgs, "include", "-I")
	if cppflags != "" {
		env = appendOrUpdateEnv(env, "CPPFLAGS", cppflags)
	}

	pkgConfigPath := fmt.Sprintf("%s/share/pkgconfig", brewPrefix)
	env = appendOrUpdateEnv(env, "PKG_CONFIG_PATH", pkgConfigPath)

	return env
}

// buildFlagString builds a flag string for the given packages.
func (m *Manager) buildFlagString(brewPrefix string, pkgs []string, subdir, flag string) string {
	var result string
	for _, pkg := range pkgs {
		pkgPath := fmt.Sprintf("%s/opt/%s/%s", brewPrefix, pkg, subdir)
		if m.fs.IsDir(pkgPath) {
			result += fmt.Sprintf(" %s%s", flag, pkgPath)
		}
	}
	return result
}

// addGmake sets CT_MAKE_FOR_BUILD to gmake if available.
func (m *Manager) addGmake(env []string, brewPrefix string) []string {
	gmakePath := fmt.Sprintf("%s/bin/gmake", brewPrefix)
	if m.fs.Exists(gmakePath) {
		env = append(env, "CT_MAKE_FOR_BUILD="+gmakePath)
	}
	return env
}

// buildPathEnv builds the PATH environment variable.
func (m *Manager) buildPathEnv(env []string, localBin, brewPrefix string) []string {
	currentPath := os.Getenv("PATH")
	newPath := fmt.Sprintf("%s:%s/opt/bison/bin:%s/opt/flex/bin:%s/opt/coreutils/libexec/gnubin:%s/opt/gnu-tar/libexec/gnubin:%s/opt/gnu-sed/libexec/gnubin:%s/opt/libtool/libexec/gnubin:%s/opt/grep/libexec/gnubin:%s/opt/gawk/libexec/gnubin:%s",
		localBin, brewPrefix, brewPrefix, brewPrefix, brewPrefix, brewPrefix, brewPrefix, brewPrefix, brewPrefix, currentPath)

	for i, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			env[i] = "PATH=" + newPath
			return env
		}
	}
	return append(env, "PATH="+newPath)
}

// appendOrUpdateEnv appends to an existing env var or creates it
func appendOrUpdateEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			env[i] = e + value
			return env
		}
	}
	return append(env, key+"="+value)
}
