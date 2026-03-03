// Package config provides configuration management for elmos.
// This file contains configuration saving and profile management.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// saveConfig is the serialised form of the workspace configuration.
// Only the fields that are written to elmos.yaml are included here;
// computed/runtime fields (Machines, CurrentMachine, Plugins, etc.) are omitted.
type saveConfig struct {
	Image    ImageConfig              `yaml:"image"`
	Build    BuildConfig              `yaml:"build"`
	QEMU     QEMUConfig               `yaml:"qemu"`
	Paths    PathsConfig              `yaml:"paths,omitempty"`
	Profiles map[string]ProfileConfig `yaml:"profiles,omitempty"`
}

// Save saves the configuration to a YAML file.
// Keys in the output match the mapstructure tags so the file can be
// loaded back by config.Load without any value loss.
func (cfg *Config) Save(path string) error {
	defaults := cfg.computeDefaults()
	prepared := cfg.prepareForSave(defaults)

	sc := saveConfig{
		Image:    prepared.Image,
		Build:    prepared.Build,
		QEMU:     prepared.QEMU,
		Paths:    prepared.Paths,
		Profiles: prepared.Profiles,
	}

	data, err := yaml.Marshal(sc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// computeDefaults calculates default config for comparison.
func (cfg *Config) computeDefaults() *Config {
	defaults := &Config{}
	if cfg.Paths.ProjectRoot != "" {
		defaults.Paths.ProjectRoot = cfg.Paths.ProjectRoot
	}
	applyComputedDefaults(defaults)
	return defaults
}

// prepareForSave creates a copy with default values cleared.
func (cfg *Config) prepareForSave(defaults *Config) Config {
	saveCfg := *cfg
	saveCfg.Paths = clearDefaultPaths(cfg.Paths, defaults.Paths)
	saveCfg.Image = clearDefaultImage(cfg.Image, defaults.Image)
	return saveCfg
}

// clearDefaultPaths clears path values that match defaults.
// ProjectRoot is ALWAYS saved so paths resolve correctly even when the
// config is loaded from a different working directory (e.g. via --config).
func clearDefaultPaths(paths, defaults PathsConfig) PathsConfig {
	result := paths
	// ProjectRoot is intentionally never cleared — it anchors all
	// workspace-relative path computations.
	if paths.KernelDir == defaults.KernelDir {
		result.KernelDir = ""
	}
	if paths.ModulesDir == defaults.ModulesDir {
		result.ModulesDir = ""
	}
	if paths.AppsDir == defaults.AppsDir {
		result.AppsDir = ""
	}
	if paths.LibrariesDir == defaults.LibrariesDir {
		result.LibrariesDir = ""
	}
	if paths.PatchesDir == defaults.PatchesDir {
		result.PatchesDir = ""
	}
	if paths.RootfsDir == defaults.RootfsDir {
		result.RootfsDir = ""
	}
	if paths.DiskImage == defaults.DiskImage {
		result.DiskImage = ""
	}
	return result
}

// clearDefaultImage clears image values that match defaults.
func clearDefaultImage(image, defaults ImageConfig) ImageConfig {
	result := image
	if image.Path == defaults.Path {
		result.Path = ""
	}
	if image.MountPoint == defaults.MountPoint {
		result.MountPoint = ""
	}
	return result
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}
	return nil
}

// ApplyProfile applies a named profile to the current configuration.
func (cfg *Config) ApplyProfile(name string) error {
	profile, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile not found: %s", name)
	}

	if profile.Arch != "" {
		cfg.Build.Arch = profile.Arch
	}
	if profile.Jobs > 0 {
		cfg.Build.Jobs = profile.Jobs
	}
	if profile.Memory != "" {
		cfg.QEMU.Memory = profile.Memory
	}
	if profile.CrossCompile != "" {
		cfg.Build.CrossCompile = profile.CrossCompile
	}

	return nil
}

// GetArchConfig returns the architecture configuration for the current build arch.
func (cfg *Config) GetArchConfig() *ArchConfig {
	return GetArchConfig(cfg.Build.Arch)
}
