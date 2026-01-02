// Package config provides configuration management for elmos.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
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
	configOnce     sync.Once
	configInstance *Config
)

// Load loads configuration from files and environment.
// This is the preferred method for new code using dependency injection.
func Load() (*Config, error) {
	var loadErr error

	configOnce.Do(func() {
		v := viper.New()

		// Set config name and paths
		v.SetConfigName("elmos")
		v.SetConfigType("yaml")

		// Config search paths
		v.AddConfigPath(".")                                                  // Current directory
		v.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".config", "elmos")) // User config
		v.AddConfigPath("/etc/elmos")                                         // System config

		// Environment variables
		v.SetEnvPrefix("ELMOS")
		v.AutomaticEnv()

		// Set defaults
		setDefaults(v)

		// Read config file (ignore errors - use defaults)
		_ = v.ReadInConfig()

		// Unmarshal into struct
		cfg := &Config{}
		if err := v.Unmarshal(cfg); err != nil {
			// Even if unmarshal fails, continue with defaults
			cfg = &Config{}
		}

		// Apply computed defaults
		applyComputedDefaults(cfg)

		configInstance = cfg
	})

	return configInstance, loadErr
}

// Get returns the current configuration.
// If Load() hasn't been called, it will be called automatically.
func Get() *Config {
	if configInstance == nil {
		cfg, _ := Load()
		return cfg
	}
	return configInstance
}

// Reset clears the cached configuration (for testing).
func Reset() {
	configOnce = sync.Once{}
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

	// Set all values
	v.Set("image", cfg.Image)
	v.Set("build", cfg.Build)
	v.Set("qemu", cfg.QEMU)
	v.Set("paths", cfg.Paths)
	v.Set("profiles", cfg.Profiles)

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
