// Package builtin provides built-in plugins for the ELMOS system.
package builtin

import (
	"context"
	"fmt"
	"time"

	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/builder"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
	"github.com/NguyenTrongPhuc552003/elmos/core/ui"
)

// KernelBuilderPlugin orchestrates kernel building and compilation.
// It wraps the v1.0 KernelBuilder and provides plugin hooks.
type KernelBuilderPlugin struct {
	builder *builder.KernelBuilder
	printer *ui.Printer
	config  *KernelBuilderConfig
}

// KernelBuilderConfig holds KernelBuilder plugin configuration.
type KernelBuilderConfig struct {
	// Parallel jobs for make -j
	Jobs int `mapstructure:"jobs"`

	// Use LLVM/Clang instead of GCC
	LLVM bool `mapstructure:"llvm"`

	// Cross-compile prefix
	CrossCompile string `mapstructure:"cross_compile"`

	// Generate compile_commands.json for IDE integration
	CompileCommands bool `mapstructure:"compile_commands"`

	// Timeout for build operations in seconds
	Timeout int `mapstructure:"timeout"`
}

// NewKernelBuilderPlugin creates a new KernelBuilder plugin instance.
func NewKernelBuilderPlugin() (*KernelBuilderPlugin, error) {
	kbp := &KernelBuilderPlugin{
		builder: nil, // Will be set during Init
		printer: nil, // Will be set during Init
		config: &KernelBuilderConfig{
			Jobs:            8, // Default to 8 parallel jobs
			LLVM:            true,
			CrossCompile:    "llvm-",
			CompileCommands: true,
			Timeout:         3600, // 1 hour default timeout
		},
	}

	return kbp, nil
}

// Name returns the plugin name.
func (kbp *KernelBuilderPlugin) Name() string {
	return "kernel-builder"
}

// Version returns the plugin version.
func (kbp *KernelBuilderPlugin) Version() string {
	return "2.0.0-alpha"
}

// Description returns the plugin description.
func (kbp *KernelBuilderPlugin) Description() string {
	return "Kernel building and compilation orchestration"
}

// Init initializes the KernelBuilder plugin.
func (kbp *KernelBuilderPlugin) Init(ctx *elcontext.Context, config map[string]interface{}) error {
	// Parse plugin configuration
	if config != nil {
		if jobs, ok := config["jobs"].(float64); ok {
			kbp.config.Jobs = int(jobs)
		}
		if llvm, ok := config["llvm"].(bool); ok {
			kbp.config.LLVM = llvm
		}
		if crossCompile, ok := config["cross_compile"].(string); ok {
			kbp.config.CrossCompile = crossCompile
		}
		if compileCommands, ok := config["compile_commands"].(bool); ok {
			kbp.config.CompileCommands = compileCommands
		}
		if timeout, ok := config["timeout"].(float64); ok {
			kbp.config.Timeout = int(timeout)
		}
	}

	// Initialize v1.0 KernelBuilder (will be injected during app startup)
	// For now, we just prepare the configuration
	return nil
}

// Validate validates the KernelBuilder configuration.
func (kbp *KernelBuilderPlugin) Validate() error {
	// Validate job count
	if kbp.config.Jobs < 1 {
		return fmt.Errorf("kernel builder jobs must be >= 1, got %d", kbp.config.Jobs)
	}

	if kbp.config.Jobs > 256 {
		return fmt.Errorf("kernel builder jobs must be <= 256 (too many), got %d", kbp.config.Jobs)
	}

	// Validate timeout
	if kbp.config.Timeout < 60 {
		return fmt.Errorf("kernel builder timeout must be >= 60 seconds, got %d", kbp.config.Timeout)
	}

	return nil
}

// Cleanup performs cleanup tasks.
func (kbp *KernelBuilderPlugin) Cleanup() error {
	// Clean up any temporary build artifacts if needed
	return nil
}

// Hooks returns the hooks this plugin registers.
func (kbp *KernelBuilderPlugin) Hooks() []plugin.HookRegistration {
	return []plugin.HookRegistration{
		{
			Event:    plugin.PreKernelBuild,
			Handler:  kbp.onPreKernelBuild,
			Priority: 5,
			Required: true,
		},
		{
			Event:    plugin.PostKernelBuild,
			Handler:  kbp.onPostKernelBuild,
			Priority: 5,
			Required: false,
		},
		{
			Event:    plugin.AfterConfigLoad,
			Handler:  kbp.onAfterConfigLoad,
			Priority: 3,
			Required: false,
		},
	}
}

// onAfterConfigLoad is called after configuration is loaded.
// It prepares the kernel builder with validated configuration.
func (kbp *KernelBuilderPlugin) onAfterConfigLoad(ctx context.Context, evt *plugin.Event) error {
	// Store metadata about kernel preparation
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}

	evt.Metadata[plugin.MetadataArchitecture] = "arm64"    // From current config
	evt.Metadata[plugin.MetadataKernelVersion] = "unknown" // Will be determined during build
	evt.Metadata[plugin.MetadataStartTime] = time.Now().Unix()

	return nil
}

// onPreKernelBuild is called before kernel build starts.
// Validates prerequisites and logs build information.
func (kbp *KernelBuilderPlugin) onPreKernelBuild(ctx context.Context, evt *plugin.Event) error {
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}

	// Record build start time
	evt.Metadata[plugin.MetadataStartTime] = time.Now().Unix()

	// Log build configuration
	evt.Metadata["kernel.jobs"] = kbp.config.Jobs
	evt.Metadata["kernel.llvm"] = kbp.config.LLVM
	evt.Metadata["kernel.cross_compile"] = kbp.config.CrossCompile

	// Validate kernel source exists
	// (actual validation will be done by v1.0 KernelBuilder)

	return nil
}

// onPostKernelBuild is called after kernel build completes.
// Collects artifacts and timing information.
func (kbp *KernelBuilderPlugin) onPostKernelBuild(ctx context.Context, evt *plugin.Event) error {
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}

	// Record build end time
	evt.Metadata[plugin.MetadataEndTime] = time.Now().Unix()

	// Calculate duration
	startTime, ok := evt.Metadata[plugin.MetadataStartTime].(int64)
	if ok {
		endTime := time.Now().Unix()
		duration := endTime - startTime
		evt.Metadata[plugin.MetadataDuration] = duration
	}

	// Artifacts will be recorded by the builder itself
	// (e.g., kernel.artifact_paths containing [Image, dtbs, modules])

	return nil
}

// SetKernelBuilder sets the v1.0 KernelBuilder instance.
// This is called during app initialization to inject the actual builder.
func (kbp *KernelBuilderPlugin) SetKernelBuilder(builder *builder.KernelBuilder) {
	kbp.builder = builder
}

// SetPrinter sets the UI printer for logging.
// This is called during app initialization.
func (kbp *KernelBuilderPlugin) SetPrinter(printer *ui.Printer) {
	kbp.printer = printer
}

// GetConfig returns the kernel builder configuration.
func (kbp *KernelBuilderPlugin) GetConfig() *KernelBuilderConfig {
	return kbp.config
}

// ApplyConfig applies the plugin configuration to the kernel builder.
// This should be called before actually building the kernel.
func (kbp *KernelBuilderPlugin) ApplyConfig() error {
	if kbp.builder == nil {
		return fmt.Errorf("kernel builder not initialized")
	}

	// Apply configuration to v1.0 builder
	// (This would be implemented based on actual builder API)
	// For now, just validate

	return kbp.Validate()
}

// RunTask implements plugin.TaskRunner.
// It delegates kernel build work to the wrapped v1.0 KernelBuilder.
func (kbp *KernelBuilderPlugin) RunTask(ctx context.Context, taskID string, config map[string]any) error {
	if kbp.builder == nil {
		return fmt.Errorf("kernel-builder: builder not initialized (call SetKernelBuilder first)")
	}

	jobs := kbp.config.Jobs
	if j, ok := config["jobs"].(int); ok && j > 0 {
		jobs = j
	}

	return kbp.builder.Build(ctx, builder.BuildOptions{Jobs: jobs})
}
