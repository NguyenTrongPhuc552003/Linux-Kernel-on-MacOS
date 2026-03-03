// Package config provides configuration management for elmos.
// This file contains all configuration struct definitions.
package config

import "fmt"

// Config holds the application configuration.
type Config struct {
	// ConfigFile is the path to the loaded configuration file
	ConfigFile string `yaml:"-"`

	// Image settings
	Image ImageConfig `mapstructure:"image"`

	// Build settings
	Build BuildConfig `mapstructure:"build"`

	// QEMU settings
	QEMU QEMUConfig `mapstructure:"qemu"`

	// Paths
	Paths PathsConfig `mapstructure:"paths"`

	// Profiles for different configurations
	Profiles map[string]ProfileConfig `mapstructure:"profiles"`

	// ========== NEW v2.0 FIELDS ==========

	// Machine is the current/default target machine name
	Machine string `mapstructure:"machine"`

	// Machines holds all available machine definitions (loaded from machine/*.yml)
	Machines map[string]*MachineDefinition `yaml:"-"`

	// CurrentMachine is the resolved machine definition for this build
	CurrentMachine *MachineDefinition `yaml:"-"`

	// Bootloader configuration (machine-agnostic settings)
	Bootloader BootloaderConfig `mapstructure:"bootloader"`

	// Plugins configuration for v2.0 plugin system
	Plugins PluginsConfig `mapstructure:"plugins"`
}

// ImageConfig holds disk image configuration.
type ImageConfig struct {
	Path       string `mapstructure:"path"        yaml:"path,omitempty"`
	VolumeName string `mapstructure:"volume_name" yaml:"volume_name,omitempty"`
	Size       string `mapstructure:"size"        yaml:"size,omitempty"`
	MountPoint string `mapstructure:"mount_point" yaml:"mount_point,omitempty"`
}

// BuildConfig holds kernel build configuration.
type BuildConfig struct {
	Arch         string `mapstructure:"arch"          yaml:"arch,omitempty"`
	Jobs         int    `mapstructure:"jobs"          yaml:"jobs,omitempty"`
	LLVM         bool   `mapstructure:"llvm"          yaml:"llvm"`
	CrossCompile string `mapstructure:"cross_compile" yaml:"cross_compile,omitempty"`
	Verbose      bool   `mapstructure:"verbose"       yaml:"verbose"`
}

// QEMUConfig holds QEMU configuration.
type QEMUConfig struct {
	Memory  string `mapstructure:"memory"   yaml:"memory,omitempty"`
	GDBPort int    `mapstructure:"gdb_port" yaml:"gdb_port,omitempty"`
	SSHPort int    `mapstructure:"ssh_port" yaml:"ssh_port,omitempty"`
	SMP     int    `mapstructure:"smp"      yaml:"smp,omitempty"`
}

// PathsConfig holds important paths.
type PathsConfig struct {
	ProjectRoot   string `mapstructure:"project_root"   yaml:"project_root,omitempty"`
	KernelDir     string `mapstructure:"kernel_dir"     yaml:"kernel_dir,omitempty"`
	ModulesDir    string `mapstructure:"modules_dir"    yaml:"modules_dir,omitempty"`
	AppsDir       string `mapstructure:"apps_dir"       yaml:"apps_dir,omitempty"`
	LibrariesDir  string `mapstructure:"libraries_dir"  yaml:"libraries_dir,omitempty"`
	PatchesDir    string `mapstructure:"patches_dir"    yaml:"patches_dir,omitempty"`
	RootfsDir     string `mapstructure:"rootfs_dir"     yaml:"rootfs_dir,omitempty"`
	DiskImage     string `mapstructure:"disk_image"     yaml:"disk_image,omitempty"`
	DebianMirror  string `mapstructure:"debian_mirror"  yaml:"debian_mirror,omitempty"`
	ToolchainsDir string `mapstructure:"toolchains_dir" yaml:"toolchains_dir,omitempty"`
}

// ProfileConfig holds a named configuration profile.
type ProfileConfig struct {
	Arch         string `mapstructure:"arch"          yaml:"arch,omitempty"`
	Jobs         int    `mapstructure:"jobs"          yaml:"jobs,omitempty"`
	Memory       string `mapstructure:"memory"        yaml:"memory,omitempty"`
	CrossCompile string `mapstructure:"cross_compile" yaml:"cross_compile,omitempty"`
}

// BootloaderConfig holds global bootloader settings
type BootloaderConfig struct {
	Type    string   `mapstructure:"type"`    // "u-boot" or "barebox"
	Repo    string   `mapstructure:"repo"`    // Repository URL
	Version string   `mapstructure:"version"` // Version/tag
	Patches []string `mapstructure:"patches"` // Global patches to apply
}

// PluginsConfig holds plugin configuration
type PluginsConfig struct {
	// Enabled plugins list
	Enabled []string `mapstructure:"enabled"`

	// Disabled plugins list
	Disabled []string `mapstructure:"disabled"`

	// Per-plugin configuration
	Plugins map[string]PluginEntry `mapstructure:"plugins"`

	// Plugin search paths (for user plugins)
	SearchPaths []string `mapstructure:"search_paths"`
}

// PluginEntry is a single plugin configuration entry
type PluginEntry struct {
	Type    string                 `mapstructure:"type"` // "builtin" or "user"
	Path    string                 `mapstructure:"path"` // For user plugins
	Enabled bool                   `mapstructure:"enabled"`
	Config  map[string]interface{} `mapstructure:"config"` // Plugin-specific config
}

// GetMachine returns the machine definition with the given name.
// Returns nil if machine is not found.
func (c *Config) GetMachine(name string) *MachineDefinition {
	if c.Machines == nil {
		return nil
	}
	return c.Machines[name]
}

// SetCurrentMachine sets the current machine and validates it.
func (c *Config) SetCurrentMachine(machine *MachineDefinition) error {
	if machine == nil {
		return fmt.Errorf("machine definition cannot be nil")
	}

	if err := machine.Validate(); err != nil {
		return err
	}

	c.CurrentMachine = machine
	c.Build.Arch = machine.Kernel.Arch
	c.Machine = machine.Name

	return nil
}
