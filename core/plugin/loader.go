// Package plugin provides plugin discovery and loading.
package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/ui"
)

// PluginConfig represents plugin configuration from plugins.yml
type PluginConfig struct {
	Enabled  []string               `yaml:"enabled"`
	Disabled []string               `yaml:"disabled"`
	Plugins  map[string]PluginEntry `yaml:"plugins"`
}

// PluginEntry represents a single plugin configuration entry
type PluginEntry struct {
	Type    string                 `yaml:"type"` // "builtin" or "user"
	Path    string                 `yaml:"path"` // For user plugins
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config"`
}

// Registry holds all loaded plugins
type Registry struct {
	plugins    map[string]Plugin
	executor   *HookExecutor
	printer    *ui.Printer
	searchPath []string
	verbose    bool
	ctx        *elcontext.Context // application context, injected after construction
}

// NewRegistry creates a new plugin registry.
func NewRegistry(executor *HookExecutor, printer *ui.Printer, verbose bool) *Registry {
	return &Registry{
		plugins:  make(map[string]Plugin),
		executor: executor,
		printer:  printer,
		verbose:  verbose,
		searchPath: []string{
			// User plugins directory
			filepath.Join(os.Getenv("HOME"), ".elmos", "plugins"),
			// Workspace-local plugins
			"./plugins",
		},
	}
}

// LoadBuiltins loads all builtin plugins.
// This is called automatically during App initialization.
// Builtin plugins are compiled into the binary.
func (r *Registry) LoadBuiltins(ctx *elcontext.Context, config map[string]interface{}) error {
	r.ctx = ctx

	builtinPlugins := []string{
		"kernel-builder",
		"bsp-manager",
		"bootloader-builder",
		"caching-backend",
		"rootfs-builder",
		"image-assembler",
	}

	if r.verbose {
		r.printer.Info("Loading %d builtin plugins", len(builtinPlugins))
	}

	for _, name := range builtinPlugins {
		if r.isPluginDisabled(config, name) {
			continue
		}
		if err := r.loadBuiltin(ctx, name, r.extractPluginConfig(config, name)); err != nil {
			r.printer.Error("Failed to load plugin %s: %v", name, err)
		}
	}

	return nil
}

// loadBuiltin creates and registers a single builtin plugin by name.
func (r *Registry) loadBuiltin(ctx *elcontext.Context, name string, cfg map[string]interface{}) error {
	factory, ok := builtinFactories[name]
	if !ok {
		return nil // Unknown builtin — skip silently
	}

	p, err := factory()
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return r.registerPlugin(ctx, p, cfg)
}

// isPluginDisabled returns true when the config entry for name sets enabled=false.
func (r *Registry) isPluginDisabled(config map[string]interface{}, name string) bool {
	entry, ok := config[name].(map[string]interface{})
	if !ok {
		return false
	}
	enabled, ok := entry["enabled"].(bool)
	if !ok {
		return false
	}
	if !enabled && r.verbose {
		r.printer.Warn("Plugin %s is disabled", name)
	}
	return !enabled
}

// extractPluginConfig pulls the plugin-specific config sub-map from the top-level config.
func (r *Registry) extractPluginConfig(config map[string]interface{}, name string) map[string]interface{} {
	entry, ok := config[name].(map[string]interface{})
	if !ok {
		return nil
	}
	cfg, ok := entry["config"].(map[string]interface{})
	if !ok {
		return nil
	}
	return cfg
}

// registerPlugin initializes and registers a plugin.
func (r *Registry) registerPlugin(ctx *elcontext.Context, plugin Plugin, config map[string]any) error {
	if plugin == nil {
		return fmt.Errorf("plugin is nil")
	}

	name := plugin.Name()

	// Check for duplicate
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	// Initialize plugin with the application context.
	if err := plugin.Init(ctx, config); err != nil {
		return fmt.Errorf("plugin %s init failed: %w", name, err)
	}

	// Validate plugin
	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("plugin %s validation failed: %w", name, err)
	}

	// Register hooks
	hooks := plugin.Hooks()
	for _, hook := range hooks {
		r.executor.Register(name, hook.Event, hook)
	}

	// Store plugin
	r.plugins[name] = plugin

	if r.verbose {
		r.printer.Success("Plugin loaded: %s v%s", name, plugin.Version())
	}

	return nil
}

// LoadUser loads user-defined plugins from search paths.
// User plugins must be compiled Go files or Go modules.
// For Phase 1, we'll skip this - implement in Phase 3 with plugin.so support.
func (r *Registry) LoadUser(config map[string]interface{}) error {
	// TODO: Phase 3 - implement Go plugin loading
	// For now, return nil to skip user plugins
	return nil
}

// Get retrieves a loaded plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	plugin, ok := r.plugins[name]
	return plugin, ok
}

// ListPlugins returns all loaded plugins as a slice.
func (r *Registry) ListPlugins() []Plugin {
	result := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		result = append(result, plugin)
	}
	return result
}

// GetPlugin retrieves a loaded plugin by name.
// Returns nil if the plugin is not found.
func (r *Registry) GetPlugin(name string) Plugin {
	return r.plugins[name]
}

// CleanupAll calls Cleanup on all loaded plugins.
// Should be called during application shutdown.
func (r *Registry) CleanupAll() error {
	var errors []error

	for name, plugin := range r.plugins {
		if err := plugin.Cleanup(); err != nil {
			errors = append(errors, fmt.Errorf("plugin %s cleanup failed: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// HasPlugin returns true if a plugin is loaded.
func (r *Registry) HasPlugin(name string) bool {
	_, ok := r.plugins[name]
	return ok
}

// Count returns the number of loaded plugins.
func (r *Registry) Count() int {
	return len(r.plugins)
}

// GetExecutor returns the hook executor.
func (r *Registry) GetExecutor() *HookExecutor {
	return r.executor
}

// --- Builtin Plugin Factory Functions ---

// PluginFactory is a function that creates a plugin instance
type PluginFactory func() (Plugin, error)

// builtinFactories holds the registered plugin factories
var builtinFactories = map[string]PluginFactory{}

// RegisterBuiltinFactory registers a factory function for a builtin plugin
func RegisterBuiltinFactory(name string, factory PluginFactory) {
	builtinFactories[name] = factory
}
