// Package plugin provides the plugin interface and lifecycle management for elmos.
// Plugins are the primary extension mechanism, allowing modular additions to the build pipeline.
package plugin

import (
	"context"

	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
)

// Plugin is the base interface that all plugins must implement.
// Plugins are loaded at application startup and can register hooks into the build lifecycle.
type Plugin interface {
	// Metadata methods
	Name() string
	Version() string
	Description() string

	// Lifecycle methods
	// Init is called when the plugin is loaded. config contains plugin-specific settings.
	Init(ctx *elcontext.Context, config map[string]interface{}) error

	// Validate is called after Init to verify the plugin state and dependencies.
	// Should return error if plugin cannot operate.
	Validate() error

	// Cleanup is called when the application shuts down or the plugin is unloaded.
	Cleanup() error

	// Hooks returns the list of hook registrations this plugin is interested in.
	// These hooks are executed during the build lifecycle.
	Hooks() []HookRegistration
}

// Event represents a build lifecycle event that is passed to hook handlers.
type Event struct {
	Name     string                 // Event name (e.g., "pre_kernel_build", "post_kernel_build")
	Metadata map[string]interface{} // Event-specific metadata (artifacts, timings, etc.)
}

// HookFunc is the callback signature for hook handlers.
// Returning an error will fail the build (unless the hook is marked non-fatal).
type HookFunc func(cliCtx context.Context, evt *Event) error

// HookRegistration registers a plugin's interest in a build lifecycle event.
type HookRegistration struct {
	// Event name (e.g., "pre_kernel_build", "post_kernel_build")
	Event string

	// Handler function to call when event fires
	Handler HookFunc

	// Priority determines execution order: higher values run first.
	// Valid range: 0-10, default 5. Useful for ordering multiple handlers.
	Priority int

	// Required determines if hook failure should fail the build.
	// If false, build continues even if hook fails.
	Required bool
}

// Hook event constants - standard build lifecycle events
const (
	// Pre-build events (validation phase)

	// BeforeInit fires before workspace initialization
	BeforeInit = "before_init"

	// AfterInit fires after workspace initialization
	AfterInit = "after_init"

	// AfterConfigLoad fires after config and plugins are loaded
	AfterConfigLoad = "after_config_load"

	// Kernel build events
	PreKernelClone  = "pre_kernel_clone"
	PostKernelClone = "post_kernel_clone"

	PreKernelConfig  = "pre_kernel_config"
	PostKernelConfig = "post_kernel_config"

	PreKernelBuild  = "pre_kernel_build"
	PostKernelBuild = "post_kernel_build"

	// Bootloader build events
	PreBootloaderClone  = "pre_bootloader_clone"
	PostBootloaderClone = "post_bootloader_clone"

	PreBootloaderConfig  = "pre_bootloader_config"
	PostBootloaderConfig = "post_bootloader_config"

	PreBootloaderBuild  = "pre_bootloader_build"
	PostBootloaderBuild = "post_bootloader_build"

	// Rootfs build events
	PreRootfsCreate  = "pre_rootfs_create"
	PostRootfsCreate = "post_rootfs_create"

	// Image assembly events
	PreImageAssemble  = "pre_image_assemble"
	PostImageAssemble = "post_image_assemble"

	// Caching events
	PreArtifactCache  = "pre_artifact_cache"
	PostArtifactCache = "post_artifact_cache"

	// Testing events
	PreQEMUBoot  = "pre_qemu_boot"
	PostQEMUBoot = "post_qemu_boot"

	// Error and cleanup
	OnBuildError = "on_build_error"
	OnCleanup    = "on_cleanup"
)

// TaskRunner executes a specific task type in the build pipeline.
// Each builtin plugin that performs actual build work implements this interface.
// PipelineExecutor calls RunTask() when a task has no cache hit.
type TaskRunner interface {
	RunTask(ctx context.Context, taskID string, config map[string]any) error
}

// CommonMetadata keys used in Event.Metadata across different hooks
const (
	// Artifact paths
	MetadataArtifactPath     = "artifact_path"
	MetadataArtifactPaths    = "artifact_paths"
	MetadataArtifactSize     = "artifact_size"
	MetadataArtifactChecksum = "artifact_checksum"

	// Build configuration
	MetadataMachine        = "machine"
	MetadataArchitecture   = "architecture"
	MetadataKernelVersion  = "kernel_version"
	MetadataBootloaderType = "bootloader_type"
	MetadataBootloaderVer  = "bootloader_version"

	// Timing and performance
	MetadataDuration    = "duration_seconds"
	MetadataStartTime   = "start_time"
	MetadataEndTime     = "end_time"
	MetadataCacheHit    = "cache_hit"
	MetadataFingerprint = "fingerprint"

	// Error information
	MetadataError       = "error"
	MetadataErrorReason = "error_reason"
)
