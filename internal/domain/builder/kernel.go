// Package builder provides kernel and module build orchestration for elmos.
package builder

import (
	"context"
	"fmt"
	"path/filepath"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/internal/config"
	elcontext "github.com/NguyenTrongPhuc552003/elmos/internal/context"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
)

// KernelBuilder orchestrates kernel build operations.
type KernelBuilder struct {
	exec executor.Executor
	fs   filesystem.FileSystem
	cfg  *elconfig.Config
	ctx  *elcontext.Context
}

// NewKernelBuilder creates a new KernelBuilder with the given dependencies.
func NewKernelBuilder(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config, ctx *elcontext.Context) *KernelBuilder {
	return &KernelBuilder{
		exec: exec,
		fs:   fs,
		cfg:  cfg,
		ctx:  ctx,
	}
}

// BuildOptions contains options for building the kernel.
type BuildOptions struct {
	Jobs    int
	Targets []string
}

// Build builds the kernel with the specified targets.
func (b *KernelBuilder) Build(ctx context.Context, opts BuildOptions) error {
	// Validate targets
	for _, target := range opts.Targets {
		if !elconfig.ValidBuildTargets[target] {
			return fmt.Errorf("invalid build target: %s", target)
		}
	}

	// Determine job count
	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = b.cfg.Build.Jobs
	}

	// Build make arguments
	args := []string{
		"-C", b.cfg.Paths.KernelDir,
		fmt.Sprintf("-j%d", jobs),
		fmt.Sprintf("ARCH=%s", b.cfg.Build.Arch),
		"LLVM=1",
		fmt.Sprintf("CROSS_COMPILE=%s", b.cfg.Build.CrossCompile),
	}
	args = append(args, opts.Targets...)

	// Run make with proper environment
	env := b.ctx.GetMakeEnv()
	return b.exec.RunWithEnv(ctx, env, "make", args...)
}

// Configure runs kernel configuration (menuconfig, defconfig, etc.).
func (b *KernelBuilder) Configure(ctx context.Context, configType string) error {
	// Validate config type
	valid := false
	for _, ct := range elconfig.KernelConfigTypes {
		if ct == configType {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid config type: %s", configType)
	}

	args := []string{
		"-C", b.cfg.Paths.KernelDir,
		fmt.Sprintf("ARCH=%s", b.cfg.Build.Arch),
		"LLVM=1",
		fmt.Sprintf("CROSS_COMPILE=%s", b.cfg.Build.CrossCompile),
		configType,
	}

	// Run make with proper environment
	env := b.ctx.GetMakeEnv()
	return b.exec.RunWithEnv(ctx, env, "make", args...)
}

// EnableKVMConfig enables KVM-specific kernel config options.
func (b *KernelBuilder) EnableKVMConfig(ctx context.Context) error {
	configFile := filepath.Join(b.cfg.Paths.KernelDir, ".config")

	if !b.fs.Exists(configFile) {
		return fmt.Errorf(".config not found - run 'elmos kernel config' first")
	}

	// Read current config
	content, err := b.fs.ReadFile(configFile)
	if err != nil {
		return err
	}

	// This is a simplified implementation
	// In practice, you'd use scripts/config or similar
	_ = content // For now, just run olddefconfig which applies defaults

	args := []string{
		"-C", b.cfg.Paths.KernelDir,
		fmt.Sprintf("ARCH=%s", b.cfg.Build.Arch),
		"LLVM=1",
		"olddefconfig",
	}

	env := b.ctx.GetMakeEnv()
	return b.exec.RunWithEnv(ctx, env, "make", args...)
}

// Clean runs distclean on the kernel source.
func (b *KernelBuilder) Clean(ctx context.Context) error {
	args := []string{
		"-C", b.cfg.Paths.KernelDir,
		fmt.Sprintf("ARCH=%s", b.cfg.Build.Arch),
		"LLVM=1",
		"distclean",
	}

	env := b.ctx.GetMakeEnv()
	return b.exec.RunWithEnv(ctx, env, "make", args...)
}

// GetDefaultTargets returns the default build targets for the current architecture.
func (b *KernelBuilder) GetDefaultTargets() []string {
	return b.ctx.GetDefaultTargets()
}

// HasConfig checks if the kernel has been configured.
func (b *KernelBuilder) HasConfig() bool {
	return b.ctx.HasConfig()
}

// HasKernelImage checks if the kernel image has been built.
func (b *KernelBuilder) HasKernelImage() bool {
	return b.ctx.HasKernelImage()
}
