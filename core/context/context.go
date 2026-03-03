// Package context provides build context management for elmos.
package context

import (
	gocontext "context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/homebrew"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/platform"
)

// Context holds the current build context and state.
type Context struct {
	Config   *config.Config
	Exec     executor.Executor
	FS       filesystem.FileSystem
	Brew     *homebrew.Resolver // TODO(platform): replace with Platform.Packages() once GetLibexecBin is in interface
	Platform platform.Platform
	Verbose  bool
}

// New creates a new build context with the given dependencies.
func New(cfg *config.Config, exec executor.Executor, fs filesystem.FileSystem) *Context {
	ctx := &Context{
		Config:   cfg,
		Exec:     exec,
		FS:       fs,
		Platform: platform.CurrentWithExecutor(exec),
	}
	// Homebrew resolver is only useful on macOS.
	if runtime.GOOS == "darwin" {
		ctx.Brew = homebrew.NewResolver(exec)
	}
	return ctx
}

// IsMounted checks if the kernel volume is currently mounted.
func (ctx *Context) IsMounted() bool {
	name := ctx.Config.Image.VolumeName
	mounted, _, err := ctx.Platform.DiskImage().IsMounted(gocontext.Background(), name)
	return err == nil && mounted
}

// EnsureMounted ensures the kernel volume is mounted.
func (ctx *Context) EnsureMounted() error {
	if !ctx.IsMounted() {
		return ImageError("kernel volume not mounted", ErrNotMounted)
	}
	return nil
}

// GetActualMountPoint returns the actual mount point path of the kernel volume.
// This handles cases where the volume is mounted at a different location (e.g. " 1" suffix).
func (ctx *Context) GetActualMountPoint() (string, error) {
	// Fast path: if configured path exists
	if ctx.FS.IsDir(ctx.Config.Image.MountPoint) {
		return ctx.Config.Image.MountPoint, nil
	}
	// Ask platform layer for the actual mount point
	name := ctx.Config.Image.VolumeName
	mounted, mp, err := ctx.Platform.DiskImage().IsMounted(gocontext.Background(), name)
	if err != nil {
		return "", err
	}
	if !mounted {
		return "", fmt.Errorf("image not mounted: %s", ctx.Config.Image.Path)
	}
	return mp, nil
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

	// On macOS, prepend Homebrew tool directories to PATH so GNU tools
	// take precedence over BSD variants. On Linux, tools are already
	// the GNU versions and live on the standard PATH.
	if runtime.GOOS == "darwin" && ctx.Brew != nil {
		newPath = ctx.prependBrewToolPaths(originalPath)
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

	// Add HOSTCFLAGS (macOS needs special flags; Linux does not)
	hostcflags := ctx.buildHostCFlags()
	if hostcflags != "" {
		env = append(env, "HOSTCFLAGS="+hostcflags)
	}

	return env
}

// prependBrewToolPaths adds Homebrew tool directories to PATH (macOS only).
func (ctx *Context) prependBrewToolPaths(currentPath string) string {
	newPath := currentPath
	if gnuSed := ctx.Brew.GetLibexecBin("gnu-sed"); gnuSed != "" {
		newPath = gnuSed + string(os.PathListSeparator) + newPath
	}
	if coreutils := ctx.Brew.GetLibexecBin("coreutils"); coreutils != "" {
		newPath = coreutils + string(os.PathListSeparator) + newPath
	}
	if llvmBin := ctx.Brew.GetBin("llvm"); llvmBin != "" {
		newPath = llvmBin + string(os.PathListSeparator) + newPath
	}
	if lldBin := ctx.Brew.GetBin("lld"); lldBin != "" {
		newPath = lldBin + string(os.PathListSeparator) + newPath
	}
	if e2fsBin := ctx.Brew.GetSbin("e2fsprogs"); e2fsBin != "" {
		newPath = e2fsBin + string(os.PathListSeparator) + newPath
	}
	return newPath
}

// buildHostCFlags constructs the HOSTCFLAGS for kernel builds.
// On macOS, adds Homebrew libelf include path and compatibility defines.
// On Linux, system headers are in standard paths; only add LibrariesDir if set.
func (ctx *Context) buildHostCFlags() string {
	if runtime.GOOS != "darwin" {
		if ctx.Config.Paths.LibrariesDir != "" {
			return "-I" + ctx.Config.Paths.LibrariesDir
		}
		return ""
	}

	// macOS: need Homebrew libelf + compatibility defines
	var flags []string
	if ctx.Config.Paths.LibrariesDir != "" {
		flags = append(flags, "-I"+ctx.Config.Paths.LibrariesDir)
	}
	if ctx.Brew != nil {
		if libelfInclude := ctx.Brew.GetInclude("libelf"); libelfInclude != "" {
			flags = append(flags, "-I"+libelfInclude)
		}
	}
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
