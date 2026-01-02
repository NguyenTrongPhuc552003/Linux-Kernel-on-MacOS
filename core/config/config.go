// Package config provides configuration management for elmos.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

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
}

// ImageConfig holds disk image configuration.
type ImageConfig struct {
	Path       string `mapstructure:"path"`
	VolumeName string `mapstructure:"volume_name"`
	Size       string `mapstructure:"size"`
	MountPoint string `mapstructure:"mount_point"`
}

// BuildConfig holds kernel build configuration.
type BuildConfig struct {
	Arch         string `mapstructure:"arch"`
	Jobs         int    `mapstructure:"jobs"`
	LLVM         bool   `mapstructure:"llvm"`
	CrossCompile string `mapstructure:"cross_compile"`
}

// QEMUConfig holds QEMU configuration.
type QEMUConfig struct {
	Memory  string `mapstructure:"memory"`
	GDBPort int    `mapstructure:"gdb_port"`
	SSHPort int    `mapstructure:"ssh_port"`
	SMP     int    `mapstructure:"smp"`
}

// PathsConfig holds important paths.
type PathsConfig struct {
	ProjectRoot  string `mapstructure:"project_root"`
	KernelDir    string `mapstructure:"kernel_dir"`
	ModulesDir   string `mapstructure:"modules_dir"`
	AppsDir      string `mapstructure:"apps_dir"`
	LibrariesDir string `mapstructure:"libraries_dir"`
	PatchesDir   string `mapstructure:"patches_dir"`
	RootfsDir    string `mapstructure:"rootfs_dir"`
	DiskImage    string `mapstructure:"disk_image"`
	DebianMirror string `mapstructure:"debian_mirror"`
}

// ProfileConfig holds a named configuration profile.
type ProfileConfig struct {
	Arch         string `mapstructure:"arch"`
	Jobs         int    `mapstructure:"jobs"`
	Memory       string `mapstructure:"memory"`
	CrossCompile string `mapstructure:"cross_compile"`
}

var (
	configInstance *Config
)

// Load loads configuration from files and environment.
// This is the preferred method for new code using dependency injection.
func Load(configPath string) (*Config, error) {
	var loadErr error
	var cfg *Config

	v := viper.New()

	if configPath != "" {
		// Use specific config file
		v.SetConfigFile(configPath)
	} else {
		// Set config name and search paths
		v.SetConfigName("elmos")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")                                                  // Current directory
		v.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".config", "elmos")) // User config
		v.AddConfigPath("/etc/elmos")                                         // System config
	}

	// Environment variables
	v.SetEnvPrefix("ELMOS")
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// If specific file provided, error out. Otherwise ignore if not found.
		if configPath != "" {
			loadErr = fmt.Errorf("failed to read config file %s: %w", configPath, err)
		} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// If it's not a "not found" error, report it
			loadErr = fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal into struct
	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply computed defaults
	applyComputedDefaults(cfg)

	// Save which file was used
	cfg.ConfigFile = v.ConfigFileUsed()

	// Update singleton
	configInstance = cfg
	return cfg, loadErr
}

// Get returns the current configuration.
// If Load() hasn't been called, it will be called automatically.
func Get() *Config {
	if configInstance == nil {
		cfg, _ := Load("")
		return cfg
	}
	return configInstance
}

// Reset clears the cached configuration (for testing).
func Reset() {
	configInstance = nil
}

// setDefaults sets default values for configuration.
func setDefaults(v *viper.Viper) {
	// Image defaults
	v.SetDefault("image.volume_name", DefaultVolumeName)
	v.SetDefault("image.size", DefaultImageSize)

	// Build defaults
	v.SetDefault("build.arch", DefaultArch)
	v.SetDefault("build.jobs", runtime.NumCPU())
	v.SetDefault("build.llvm", true)
	v.SetDefault("build.cross_compile", DefaultCrossPrefix)

	// QEMU defaults
	v.SetDefault("qemu.memory", DefaultMemory)
	v.SetDefault("qemu.gdb_port", DefaultGDBPort)
	v.SetDefault("qemu.ssh_port", DefaultSSHPort)
	v.SetDefault("qemu.smp", runtime.NumCPU())

	// Paths defaults
	v.SetDefault("paths.debian_mirror", DefaultDebianMirror)
}

// applyComputedDefaults fills in paths based on project root.
func applyComputedDefaults(cfg *Config) {
	// Find project root (where go.mod or elmos.yaml exists)
	if cfg.Paths.ProjectRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			cfg.Paths.ProjectRoot = cwd
		}
	}

	root := cfg.Paths.ProjectRoot

	// Image path
	if cfg.Image.Path == "" {
		cfg.Image.Path = filepath.Join(root, "img.sparseimage")
	}

	// Mount point
	if cfg.Image.MountPoint == "" {
		cfg.Image.MountPoint = filepath.Join("/Volumes", cfg.Image.VolumeName)
	}

	// Kernel directory (inside mount)
	if cfg.Paths.KernelDir == "" {
		cfg.Paths.KernelDir = filepath.Join(cfg.Image.MountPoint, "linux")
	}

	// Modules directory (project root)
	if cfg.Paths.ModulesDir == "" {
		cfg.Paths.ModulesDir = filepath.Join(root, "modules")
	}

	// Apps directory (project root)
	if cfg.Paths.AppsDir == "" {
		cfg.Paths.AppsDir = filepath.Join(root, "apps")
	}

	// Libraries directory (project root)
	if cfg.Paths.LibrariesDir == "" {
		cfg.Paths.LibrariesDir = filepath.Join(root, "libraries")
	}

	// Patches directory (project root)
	if cfg.Paths.PatchesDir == "" {
		cfg.Paths.PatchesDir = filepath.Join(root, "patches")
	}

	// Rootfs directory (inside mount)
	if cfg.Paths.RootfsDir == "" {
		cfg.Paths.RootfsDir = filepath.Join(cfg.Image.MountPoint, "rootfs")
	}

	// Disk image (inside mount)
	if cfg.Paths.DiskImage == "" {
		cfg.Paths.DiskImage = filepath.Join(cfg.Image.MountPoint, "disk.img")
	}
}

// Save saves the configuration to a YAML file.
func (cfg *Config) Save(path string) error {
	v := viper.New()
	v.SetConfigType("yaml")

	// Calculate defaults to compare against
	defaults := &Config{}
	// If project root is set in current config, use it for defaults calculation base
	if cfg.Paths.ProjectRoot != "" {
		defaults.Paths.ProjectRoot = cfg.Paths.ProjectRoot
	}
	applyComputedDefaults(defaults)

	// Create a copy to modify for saving
	saveCfg := *cfg
	savePaths := saveCfg.Paths

	// Only save paths that differ from defaults
	if savePaths.ProjectRoot == defaults.Paths.ProjectRoot {
		savePaths.ProjectRoot = ""
	}
	if savePaths.KernelDir == defaults.Paths.KernelDir {
		savePaths.KernelDir = ""
	}
	if savePaths.ModulesDir == defaults.Paths.ModulesDir {
		savePaths.ModulesDir = ""
	}
	if savePaths.AppsDir == defaults.Paths.AppsDir {
		savePaths.AppsDir = ""
	}
	if savePaths.LibrariesDir == defaults.Paths.LibrariesDir {
		savePaths.LibrariesDir = ""
	}
	if savePaths.PatchesDir == defaults.Paths.PatchesDir {
		savePaths.PatchesDir = ""
	}
	if savePaths.RootfsDir == defaults.Paths.RootfsDir {
		savePaths.RootfsDir = ""
	}
	if savePaths.DiskImage == defaults.Paths.DiskImage {
		savePaths.DiskImage = ""
	}

	// Verify image paths too
	if saveCfg.Image.Path == defaults.Image.Path {
		saveCfg.Image.Path = ""
	}
	if saveCfg.Image.MountPoint == defaults.Image.MountPoint {
		saveCfg.Image.MountPoint = ""
	}

	saveCfg.Paths = savePaths

	// Set all values
	v.Set("image", saveCfg.Image)
	v.Set("build", saveCfg.Build)
	v.Set("qemu", saveCfg.QEMU)
	v.Set("paths", saveCfg.Paths)
	v.Set("profiles", saveCfg.Profiles)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	// Write config
	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
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
