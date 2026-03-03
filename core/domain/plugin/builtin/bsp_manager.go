// Package builtin provides built-in plugins for the ELMOS system.
package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/NguyenTrongPhuc552003/elmos/core/config"
	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

// BspManagerPlugin manages board support packages and machine definitions.
// It provides machine discovery, validation, and caching.
type BspManagerPlugin struct {
	registries map[string]*BspRegistry
	cache      map[string]*config.MachineDefinition
	cachePath  string
	mu         sync.RWMutex
}

// BspRegistry represents a board support package registry.
type BspRegistry struct {
	Name      string
	URL       string
	CachePath string
}

// NewBspManagerPlugin creates a new BspManager plugin instance.
func NewBspManagerPlugin() (*BspManagerPlugin, error) {
	bm := &BspManagerPlugin{
		registries: make(map[string]*BspRegistry),
		cache:      make(map[string]*config.MachineDefinition),
	}

	// Set up cache path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	bm.cachePath = filepath.Join(homeDir, ".elmos", "bsp-cache")

	return bm, nil
}

// Name returns the plugin name.
func (bm *BspManagerPlugin) Name() string {
	return "bsp-manager"
}

// Version returns the plugin version.
func (bm *BspManagerPlugin) Version() string {
	return "2.0.0-alpha"
}

// Description returns the plugin description.
func (bm *BspManagerPlugin) Description() string {
	return "Board support package management and machine definition resolution"
}

// Init initializes the BSP manager plugin.
func (bm *BspManagerPlugin) Init(ctx *elcontext.Context, config map[string]interface{}) error {
	// Extract registries configuration if provided
	if registriesConfig, ok := config["registries"]; ok {
		if registriesList, ok := registriesConfig.([]interface{}); ok {
			for _, reg := range registriesList {
				if regMap, ok := reg.(map[string]interface{}); ok {
					if name, ok := regMap["name"].(string); ok {
						if url, ok := regMap["url"].(string); ok {
							registry := &BspRegistry{
								Name:      name,
								URL:       url,
								CachePath: filepath.Join(bm.cachePath, name),
							}
							bm.registries[name] = registry
						}
					}
				}
			}
		}
	}

	// Add default official registry if no registries configured
	if len(bm.registries) == 0 {
		defaultRegistry := &BspRegistry{
			Name:      "official",
			URL:       "https://github.com/elmos-sdk/bsp-registry",
			CachePath: filepath.Join(bm.cachePath, "official"),
		}
		bm.registries["official"] = defaultRegistry
	}

	// Create cache directories
	for _, registry := range bm.registries {
		if err := os.MkdirAll(registry.CachePath, 0755); err != nil {
			// Don't fail - cache is optional
			return nil
		}
	}

	return nil
}

// Validate validates the BSP manager configuration.
func (bm *BspManagerPlugin) Validate() error {
	if len(bm.registries) == 0 {
		return fmt.Errorf("no BSP registries configured")
	}
	return nil
}

// Cleanup performs cleanup tasks.
func (bm *BspManagerPlugin) Cleanup() error {
	bm.mu.Lock()
	bm.cache = make(map[string]*config.MachineDefinition)
	bm.mu.Unlock()
	return nil
}

// Hooks returns the hooks this plugin registers.
func (bm *BspManagerPlugin) Hooks() []plugin.HookRegistration {
	return []plugin.HookRegistration{
		{
			Event:    plugin.AfterConfigLoad,
			Handler:  bm.onAfterConfigLoad,
			Priority: 9, // High priority - runs early
			Required: false,
		},
	}
}

// onAfterConfigLoad is called after configuration is loaded.
// It validates the configured machine definition.
func (bm *BspManagerPlugin) onAfterConfigLoad(ctx context.Context, evt *plugin.Event) error {
	// Machine validation happened during config loading
	// This hook is here for future extensibility
	return nil
}

// GetMachine retrieves a machine definition by name.
// It first checks local cache, then BSP registries.
func (bm *BspManagerPlugin) GetMachine(name string) (*config.MachineDefinition, error) {
	bm.mu.RLock()
	if cached, ok := bm.cache[name]; ok {
		bm.mu.RUnlock()
		return cached, nil
	}
	bm.mu.RUnlock()

	// Search in registries (in order)
	for _, registry := range bm.registries {
		machine, err := bm.loadMachineFromRegistry(registry, name)
		if err == nil {
			bm.mu.Lock()
			bm.cache[name] = machine
			bm.mu.Unlock()
			return machine, nil
		}
	}

	return nil, fmt.Errorf("machine '%s' not found in any registry", name)
}

// loadMachineFromRegistry attempts to load a machine definition from a specific registry.
// The YAML file is expected to have a top-level "machine:" key wrapping the definition.
func (bm *BspManagerPlugin) loadMachineFromRegistry(registry *BspRegistry, machineName string) (*config.MachineDefinition, error) {
	machineFile := filepath.Join(registry.CachePath, fmt.Sprintf("%s.yml", machineName))

	data, err := os.ReadFile(machineFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("machine file not found: %s", machineFile)
		}
		return nil, fmt.Errorf("failed to read machine file %s: %w", machineFile, err)
	}

	// Machine YAML files use a top-level "machine:" envelope.
	var envelope struct {
		Machine *config.MachineDefinition `yaml:"machine"`
	}
	if err := yaml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse machine file %s: %w", machineFile, err)
	}
	if envelope.Machine == nil {
		return nil, fmt.Errorf("machine file %s: missing top-level 'machine:' key", machineFile)
	}

	machine := envelope.Machine
	if machine.Name == "" {
		machine.Name = machineName // Infer name from file if omitted
	}

	if err := bm.ValidateMachineDefinition(machine); err != nil {
		return nil, fmt.Errorf("machine file %s is invalid: %w", machineFile, err)
	}

	return machine, nil
}

// ListMachines returns all available machine names across all registries.
// For Phase 1, returns only machines that have local definitions.
func (bm *BspManagerPlugin) ListMachines() ([]string, error) {
	var machines []string
	seen := make(map[string]bool)

	// Scan each registry's cache directory
	for _, registry := range bm.registries {
		entries, err := os.ReadDir(registry.CachePath)
		if err != nil {
			// Skip if cache directory doesn't exist
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yml" {
				name := entry.Name()[:len(entry.Name())-4] // Remove .yml
				if !seen[name] {
					machines = append(machines, name)
					seen[name] = true
				}
			}
		}
	}

	return machines, nil
}

// GetPrimaryRegistry returns the primary (official) registry.
func (bm *BspManagerPlugin) GetPrimaryRegistry() *BspRegistry {
	if primary, ok := bm.registries["official"]; ok {
		return primary
	}

	// Fallback to first registry
	for _, registry := range bm.registries {
		return registry
	}

	return nil
}

// RegistryCount returns the number of configured registries.
func (bm *BspManagerPlugin) RegistryCount() int {
	return len(bm.registries)
}

// RegistryNames returns all configured registry names.
func (bm *BspManagerPlugin) RegistryNames() []string {
	names := make([]string, 0, len(bm.registries))
	for name := range bm.registries {
		names = append(names, name)
	}
	return names
}

// ValidateMachineDefinition validates that a machine definition is well-formed.
func (bm *BspManagerPlugin) ValidateMachineDefinition(machine *config.MachineDefinition) error {
	if machine == nil {
		return fmt.Errorf("machine definition is nil")
	}

	if machine.Name == "" {
		return fmt.Errorf("machine name is required")
	}

	// Validate name format (alphanumeric, underscore, hyphen only)
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(machine.Name) {
		return fmt.Errorf("invalid machine name '%s': must contain only alphanumeric, underscore, and hyphen", machine.Name)
	}

	// Validate kernel architecture is set
	if machine.Kernel.Arch == "" {
		return fmt.Errorf("machine '%s': kernel architecture must be specified", machine.Name)
	}

	// Validate kernel architecture against known architectures
	validArchs := map[string]bool{
		"x86_64":  true,
		"x86":     true,
		"arm64":   true,
		"armv7l":  true,
		"armv6l":  true,
		"riscv64": true,
		"ppc64le": true,
		"s390x":   true,
	}

	if !validArchs[machine.Kernel.Arch] {
		return fmt.Errorf("machine '%s': unsupported architecture '%s'", machine.Name, machine.Kernel.Arch)
	}

	return nil
}

// CacheMachine stores a machine definition in the cache.
func (bm *BspManagerPlugin) CacheMachine(machine *config.MachineDefinition) error {
	if err := bm.ValidateMachineDefinition(machine); err != nil {
		return err
	}

	bm.mu.Lock()
	bm.cache[machine.Name] = machine
	bm.mu.Unlock()

	return nil
}

// ClearCache removes all cached machines.
func (bm *BspManagerPlugin) ClearCache() {
	bm.mu.Lock()
	bm.cache = make(map[string]*config.MachineDefinition)
	bm.mu.Unlock()
}

// GetCachedMachine retrieves a machine from in-memory cache only.
// Returns nil if not cached.
func (bm *BspManagerPlugin) GetCachedMachine(name string) *config.MachineDefinition {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.cache[name]
}
