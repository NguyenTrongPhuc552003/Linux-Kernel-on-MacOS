// Package builtin provides built-in plugins for the ELMOS system.
// This file wraps the core/domain/rootfs package as a lifecycle-aware plugin.
package builtin

import (
	"context"
	"fmt"
	"time"

	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/rootfs"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

// RootfsBuilderPlugin wraps the rootfs.Creator and exposes rootfs
// creation as a plugin with pre/post lifecycle hooks.
type RootfsBuilderPlugin struct {
	exec   executor.Executor
	fs     filesystem.FileSystem
	appCtx *elcontext.Context
	config *RootfsBuilderConfig
}

// RootfsBuilderConfig holds configuration for the rootfs builder.
type RootfsBuilderConfig struct {
	// Distribution is the Linux distribution to use (e.g. "debian").
	Distribution string `mapstructure:"distribution"`

	// Release is the distro release codename (e.g. "bookworm").
	Release string `mapstructure:"release"`

	// ExtraPackages lists additional packages to install.
	ExtraPackages []string `mapstructure:"extra_packages"`

	// PostBuildScript is an optional path to a script run after rootfs creation.
	PostBuildScript string `mapstructure:"post_build_script"`

	// SizeGB is the rootfs image size in gigabytes.
	SizeGB int `mapstructure:"size_gb"`
}

// NewRootfsBuilderPlugin creates a new RootfsBuilder plugin with no executor
// (must be injected before building).
func NewRootfsBuilderPlugin() (*RootfsBuilderPlugin, error) {
	return &RootfsBuilderPlugin{
		config: &RootfsBuilderConfig{
			Distribution: "debian",
			Release:      "bookworm",
			SizeGB:       5,
		},
	}, nil
}

// Name returns the plugin name.
func (r *RootfsBuilderPlugin) Name() string { return "rootfs-builder" }

// Version returns the plugin version.
func (r *RootfsBuilderPlugin) Version() string { return "1.0.0" }

// Description returns the plugin description.
func (r *RootfsBuilderPlugin) Description() string {
	return "Rootfs creation via debootstrap with post-build customisation"
}

// Init stores the application context and applies plugin configuration.
func (r *RootfsBuilderPlugin) Init(ctx *elcontext.Context, cfg map[string]interface{}) error {
	r.appCtx = ctx
	if cfg == nil {
		return nil
	}
	if dist, ok := cfg["distribution"].(string); ok {
		r.config.Distribution = dist
	}
	if release, ok := cfg["release"].(string); ok {
		r.config.Release = release
	}
	if size, ok := cfg["size_gb"].(float64); ok && size > 0 {
		r.config.SizeGB = int(size)
	}
	if script, ok := cfg["post_build_script"].(string); ok {
		r.config.PostBuildScript = script
	}
	return nil
}

// Validate checks that required configuration is present.
func (r *RootfsBuilderPlugin) Validate() error {
	if r.config.Distribution == "" {
		return fmt.Errorf("rootfs-builder: distribution is required")
	}
	if r.config.Release == "" {
		return fmt.Errorf("rootfs-builder: release is required")
	}
	if r.config.SizeGB < 1 {
		return fmt.Errorf("rootfs-builder: size_gb must be >= 1")
	}
	return nil
}

// Cleanup is a no-op.
func (r *RootfsBuilderPlugin) Cleanup() error { return nil }

// Hooks registers pre/post rootfs lifecycle hooks.
func (r *RootfsBuilderPlugin) Hooks() []plugin.HookRegistration {
	return []plugin.HookRegistration{
		{
			Event:    plugin.PreRootfsCreate,
			Handler:  r.onPreCreate,
			Priority: 5,
			Required: true,
		},
		{
			Event:    plugin.PostRootfsCreate,
			Handler:  r.onPostCreate,
			Priority: 5,
			Required: false,
		},
	}
}

// onPreCreate validates executor availability and records start time.
func (r *RootfsBuilderPlugin) onPreCreate(_ context.Context, evt *plugin.Event) error {
	if r.exec == nil {
		return fmt.Errorf("rootfs-builder: executor not initialised")
	}
	if r.appCtx == nil {
		return fmt.Errorf("rootfs-builder: application context not available")
	}
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}
	evt.Metadata[plugin.MetadataStartTime] = time.Now().Unix()
	evt.Metadata["rootfs.distribution"] = r.config.Distribution
	evt.Metadata["rootfs.release"] = r.config.Release
	return nil
}

// onPostCreate records duration and lists produced artifacts.
func (r *RootfsBuilderPlugin) onPostCreate(_ context.Context, evt *plugin.Event) error {
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}
	evt.Metadata[plugin.MetadataEndTime] = time.Now().Unix()
	if start, ok := evt.Metadata[plugin.MetadataStartTime].(int64); ok {
		evt.Metadata[plugin.MetadataDuration] = time.Now().Unix() - start
	}
	return nil
}

// CreateRootfs executes the rootfs creation using the injected executor.
// This is called directly by the PipelineExecutor (not via hooks).
func (r *RootfsBuilderPlugin) CreateRootfs(ctx context.Context) error {
	if r.exec == nil {
		return fmt.Errorf("rootfs-builder: executor not injected")
	}
	if r.appCtx == nil || r.appCtx.Config == nil {
		return fmt.Errorf("rootfs-builder: application context not available")
	}

	creator := rootfs.NewCreator(r.exec, r.fs, r.appCtx.Config)
	opts := rootfs.CreateOptions{
		Size: fmt.Sprintf("%dG", r.config.SizeGB),
	}
	return creator.Create(ctx, opts)
}

// SetExecutor injects the command executor. Called during app initialisation.
func (r *RootfsBuilderPlugin) SetExecutor(exec executor.Executor) {
	r.exec = exec
}

// SetFilesystem injects the filesystem abstraction.
func (r *RootfsBuilderPlugin) SetFilesystem(fs filesystem.FileSystem) {
	r.fs = fs
}

// GetConfig returns the current rootfs builder configuration.
func (r *RootfsBuilderPlugin) GetConfig() *RootfsBuilderConfig {
	return r.config
}

// RunTask implements plugin.TaskRunner.
// It delegates rootfs creation work to CreateRootfs.
func (r *RootfsBuilderPlugin) RunTask(ctx context.Context, taskID string, config map[string]any) error {
	return r.CreateRootfs(ctx)
}
