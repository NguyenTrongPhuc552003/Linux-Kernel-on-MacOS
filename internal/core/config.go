// Package core provides core types, configuration, and context for elmos.
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

// Default values
const (
	DefaultImageSize   = "20G"
	DefaultVolumeName  = "kernel-dev"
	DefaultArch        = "arm64"
	DefaultCrossPrefix = "llvm-"
	DefaultMemory      = "2G"
	DefaultGDBPort     = 1234
	DefaultDebianMirror = "http://deb.debian.org/debian"
)

// Config holds the application configuration
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

// ImageConfig holds disk image configuration
type ImageConfig struct {
	Path       string `mapstructure:"path"`
	VolumeName string `mapstructure:"volume_name"`
	Size       string `mapstructure:"size"`
	MountPoint string `mapstructure:"mount_point"`
}

// BuildConfig holds kernel build configuration
type BuildConfig struct {
	Arch         string `mapstructure:"arch"`
	Jobs         int    `mapstructure:"jobs"`
	LLVM         bool   `mapstructure:"llvm"`
	CrossCompile string `mapstructure:"cross_compile"`
}

// QEMUConfig holds QEMU configuration
type QEMUConfig struct {
	Memory  string `mapstructure:"memory"`
	GDBPort int    `mapstructure:"gdb_port"`
	SMP     int    `mapstructure:"smp"`
}

// PathsConfig holds important paths
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

// ProfileConfig holds a named configuration profile
type ProfileConfig struct {
	Arch         string `mapstructure:"arch"`
	Jobs         int    `mapstructure:"jobs"`
	Memory       string `mapstructure:"memory"`
	CrossCompile string `mapstructure:"cross_compile"`
}

// configInstance is the global configuration
var configInstance *Config

// LoadConfig loads configuration from files and environment
func LoadConfig() (*Config, error) {
	if configInstance != nil {
		return configInstance, nil
	}

	v := viper.New()

	// Set config name and paths
	v.SetConfigName("elmos")
	v.SetConfigType("yaml")

	// Config search paths
	v.AddConfigPath(".")                                     // Current directory
	v.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".config", "elmos")) // User config
	v.AddConfigPath("/etc/elmos")                            // System config

	// Environment variables
	v.SetEnvPrefix("ELMOS")
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Read config file (ignore all errors - just use defaults)
	// This prevents issues with other files being picked up
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
	return cfg, nil
}

// GetConfig returns the current configuration (must call LoadConfig first)
func GetConfig() *Config {
	if configInstance == nil {
		cfg, _ := LoadConfig()
		return cfg
	}
	return configInstance
}

// setDefaults sets default values for configuration
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
	v.SetDefault("qemu.smp", runtime.NumCPU())

	// Paths defaults
	v.SetDefault("paths.debian_mirror", DefaultDebianMirror)
}

// applyComputedDefaults fills in paths based on project root
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

// SaveConfig saves the current configuration to a YAML file
func SaveConfig(cfg *Config, path string) error {
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
		return ConfigError(fmt.Sprintf("failed to create config directory: %s", dir), err)
	}

	// Write config
	if err := v.WriteConfigAs(path); err != nil {
		return ConfigError("failed to write config file", err)
	}

	return nil
}

// ApplyProfile applies a named profile to the current configuration
func (cfg *Config) ApplyProfile(name string) error {
	profile, ok := cfg.Profiles[name]
	if !ok {
		return ConfigError(fmt.Sprintf("profile not found: %s", name), nil)
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
