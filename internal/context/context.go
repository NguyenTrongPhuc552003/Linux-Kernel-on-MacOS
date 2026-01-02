// Package context provides build context management for elmos.
package context

import (
	gocontext "context"
	"os"
	"path/filepath"
	"strings"

	"github.com/NguyenTrongPhuc552003/elmos/internal/config"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/homebrew"
)

// Context holds the current build context and state.
type Context struct {
	Config  *config.Config
	Exec    executor.Executor
	FS      filesystem.FileSystem
	Brew    *homebrew.Resolver
	Verbose bool
}

// New creates a new build context with the given dependencies.
func New(cfg *config.Config, exec executor.Executor, fs filesystem.FileSystem) *Context {
	return &Context{
		Config: cfg,
		Exec:   exec,
		FS:     fs,
		Brew:   homebrew.NewResolver(exec),
	}
}

// IsMounted checks if the kernel volume is currently mounted.
func (ctx *Context) IsMounted() bool {
	mountPoint := ctx.Config.Image.MountPoint

	// Check if mount point exists and is a directory
	if !ctx.FS.IsDir(mountPoint) {
		return false
	}

	// Use mount command to verify
	out, err := ctx.Exec.Output(gocontext.Background(), "mount")
	if err != nil {
		return false
	}

	return strings.Contains(string(out), mountPoint)
}

// EnsureMounted ensures the kernel volume is mounted.
func (ctx *Context) EnsureMounted() error {
	if !ctx.IsMounted() {
		return ImageError("kernel volume not mounted", ErrNotMounted)
	}
	return nil
}

// KernelExists checks if the kernel source directory exists.
func (ctx *Context) KernelExists() bool {
	gitDir := filepath.Join(ctx.Config.Paths.KernelDir, ".git")
	return ctx.FS.Exists(gitDir)
}

// HasConfig checks if the kernel has been configured (.config exists).
func (ctx *Context) HasConfig() bool {
	configFile := filepath.Join(ctx.Config.Paths.KernelDir, ".config")
	return ctx.FS.Exists(configFile)
}

// GetKernelImage returns the path to the built kernel image for the current arch.
func (ctx *Context) GetKernelImage() string {
	archCfg := ctx.Config.GetArchConfig()
	if archCfg == nil {
		return ""
	}
	return filepath.Join(ctx.Config.Paths.KernelDir, "arch", archCfg.KernelArch, "boot", archCfg.KernelImage)
}

// GetVmlinux returns the path to vmlinux (for debugging).
func (ctx *Context) GetVmlinux() string {
	return filepath.Join(ctx.Config.Paths.KernelDir, "vmlinux")
}

// HasKernelImage checks if the kernel image has been built.
func (ctx *Context) HasKernelImage() bool {
	return ctx.FS.Exists(ctx.GetKernelImage())
}

// GetMakeEnv returns environment variables for kernel make commands.
func (ctx *Context) GetMakeEnv() []string {
	cfg := ctx.Config

	var env []string
	originalPath := os.Getenv("PATH")
	newPath := originalPath

	// Prepend GNU tools to PATH
	if gnuSed := ctx.Brew.GetLibexecBin("gnu-sed"); gnuSed != "" {
		newPath = gnuSed + string(os.PathListSeparator) + newPath
	}
	if coreutils := ctx.Brew.GetLibexecBin("coreutils"); coreutils != "" {
		newPath = coreutils + string(os.PathListSeparator) + newPath
	}

	// LLVM toolchain
	if llvmBin := ctx.Brew.GetBin("llvm"); llvmBin != "" {
		newPath = llvmBin + string(os.PathListSeparator) + newPath
	}
	if lldBin := ctx.Brew.GetBin("lld"); lldBin != "" {
		newPath = lldBin + string(os.PathListSeparator) + newPath
	}

	// e2fsprogs (sbin)
	if e2fsBin := ctx.Brew.GetSbin("e2fsprogs"); e2fsBin != "" {
		newPath = e2fsBin + string(os.PathListSeparator) + newPath
	}

	// Reconstruct env, skipping original PATH
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "PATH=") {
			env = append(env, e)
		}
	}
	env = append(env, "PATH="+newPath)

	// Add build-specific environment
	env = append(env,
		"ARCH="+cfg.Build.Arch,
		"LLVM=1",
		"CROSS_COMPILE="+cfg.Build.CrossCompile,
	)

	// Add HOSTCFLAGS for macOS compatibility
	hostcflags := ctx.buildHostCFlags()
	if hostcflags != "" {
		env = append(env, "HOSTCFLAGS="+hostcflags)
	}

	return env
}

// buildHostCFlags constructs the HOSTCFLAGS for macOS kernel builds.
func (ctx *Context) buildHostCFlags() string {
	var flags []string

	// Custom macOS headers
	if ctx.Config.Paths.LibrariesDir != "" {
		flags = append(flags, "-I"+ctx.Config.Paths.LibrariesDir)
	}

	// libelf include path (from Homebrew)
	if libelfInclude := ctx.Brew.GetInclude("libelf"); libelfInclude != "" {
		flags = append(flags, "-I"+libelfInclude)
	}

	// macOS compatibility flags
	flags = append(flags,
		"-D_UUID_T",
		"-D__GETHOSTUUID_H",
		"-D_DARWIN_C_SOURCE",
		"-D_FILE_OFFSET_BITS=64",
	)

	return strings.Join(flags, " ")
}

// GetDefaultTargets returns the default build targets for the current architecture.
func (ctx *Context) GetDefaultTargets() []string {
	archCfg := ctx.Config.GetArchConfig()
	if archCfg == nil {
		return []string{"Image", "dtbs", "modules"}
	}
	return archCfg.DefaultTargets
}
