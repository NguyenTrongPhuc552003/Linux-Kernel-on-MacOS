// Package app provides the CLI application layer for elmos.
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/internal/config"
	elcontext "github.com/NguyenTrongPhuc552003/elmos/internal/context"
	"github.com/NguyenTrongPhuc552003/elmos/internal/domain/builder"
	"github.com/NguyenTrongPhuc552003/elmos/internal/domain/doctor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/domain/emulator"
	"github.com/NguyenTrongPhuc552003/elmos/internal/domain/patch"
	"github.com/NguyenTrongPhuc552003/elmos/internal/domain/rootfs"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/internal/ui"
	"github.com/NguyenTrongPhuc552003/elmos/internal/ui/tui"
	"github.com/NguyenTrongPhuc552003/elmos/pkg/version"
)

// App holds all the application dependencies.
type App struct {
	Exec          executor.Executor
	FS            filesystem.FileSystem
	Config        *config.Config
	Context       *elcontext.Context
	KernelBuilder *builder.KernelBuilder
	ModuleBuilder *builder.ModuleBuilder
	AppBuilder    *builder.AppBuilder
	QEMURunner    *emulator.QEMURunner
	HealthChecker *doctor.HealthChecker
	AutoFixer     *doctor.AutoFixer
	RootfsCreator *rootfs.Creator
	PatchManager  *patch.Manager
	Printer       *ui.Printer
	Verbose       bool
	Interactive   bool
}

// New creates a new App with all dependencies wired up.
func New(exec executor.Executor, fs filesystem.FileSystem, cfg *config.Config) *App {
	ctx := elcontext.New(cfg, exec, fs)
	return &App{
		Exec:          exec,
		FS:            fs,
		Config:        cfg,
		Context:       ctx,
		KernelBuilder: builder.NewKernelBuilder(exec, fs, cfg, ctx),
		ModuleBuilder: builder.NewModuleBuilder(exec, fs, cfg, ctx),
		AppBuilder:    builder.NewAppBuilder(exec, fs, cfg),
		QEMURunner:    emulator.NewQEMURunner(exec, fs, cfg, ctx),
		HealthChecker: doctor.NewHealthChecker(exec, fs, cfg),
		AutoFixer:     doctor.NewAutoFixer(fs, cfg),
		RootfsCreator: rootfs.NewCreator(exec, fs, cfg),
		PatchManager:  patch.NewManager(exec, fs, cfg),
		Printer:       ui.NewPrinter(),
	}
}

// BuildRootCommand builds the root cobra command with all subcommands.
func (a *App) BuildRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "elmos",
		Short: "Embedded Linux on MacOS - Native kernel build tools",
		Long: `ELMOS (Embedded Linux on MacOS) provides native Linux kernel build tools 
for macOS, targeting RISC-V, ARM64, and more architectures.

Common workflow:
  elmos init                # Initialize workspace (mount volume, clone kernel)
  elmos doctor              # Check dependencies
  elmos config set arch arm64  # Set target architecture
  elmos kernel config       # Configure kernel
  elmos build               # Build kernel
  elmos qemu run            # Test in QEMU
  elmos tui                 # Launch interactive TUI`,
		Version: version.Get().String(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "tui" || cmd.Name() == "init" {
				return nil
			}
			a.Context.Verbose = a.Verbose
			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&a.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&a.Interactive, "interactive", "i", false, "enable interactive TUI mode")

	rootCmd.AddCommand(a.buildVersionCommand())
	rootCmd.AddCommand(a.buildTUICommand())
	rootCmd.AddCommand(a.buildInitCommand())
	rootCmd.AddCommand(a.buildConfigCommand())
	rootCmd.AddCommand(a.buildDoctorCommand())
	rootCmd.AddCommand(a.buildBuildCommand())
	rootCmd.AddCommand(a.buildKernelCommand())
	rootCmd.AddCommand(a.buildModuleCommand())
	rootCmd.AddCommand(a.buildAppsCommand())
	rootCmd.AddCommand(a.buildQEMUCommand())
	rootCmd.AddCommand(a.buildRootfsCommand())
	rootCmd.AddCommand(a.buildPatchCommand())

	return rootCmd
}

func (a *App) buildVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.Get()
			a.Printer.Print("ELMOS - Embedded Linux on MacOS")
			a.Printer.Print("Version:    %s", ui.AccentStyle.Render(info.Version))
			a.Printer.Print("Commit:     %s", info.Commit)
			a.Printer.Print("Built:      %s", info.BuildDate)
			a.Printer.Print("Go version: %s", info.GoVersion)
			a.Printer.Print("OS/Arch:    %s/%s", info.OS, info.Arch)
		},
	}
}

func (a *App) buildTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive Text User Interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}
}

func (a *App) buildInitCommand() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize workspace (mount volume and clone kernel)",
		Long: `Initialize the ELMOS workspace by:
1. Creating a sparse disk image if it doesn't exist
2. Mounting the volume
3. Optionally cloning the Linux kernel source`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.initWorkspace(cmd)
		},
	}

	mountCmd := &cobra.Command{
		Use:   "mount",
		Short: "Mount the sparse disk image",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.mountVolume(cmd)
		},
	}

	unmountCmd := &cobra.Command{
		Use:   "unmount",
		Short: "Unmount the sparse disk image",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.unmountVolume(cmd)
		},
	}

	cloneCmd := &cobra.Command{
		Use:   "clone [git-url]",
		Short: "Clone the Linux kernel source",
		Long:  `Clone the Linux kernel source into the mounted volume. Defaults to torvalds/linux.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			url := "https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git"
			if len(args) > 0 {
				url = args[0]
			}
			return a.cloneKernel(cmd, url)
		},
	}

	initCmd.AddCommand(mountCmd)
	initCmd.AddCommand(unmountCmd)
	initCmd.AddCommand(cloneCmd)
	return initCmd
}

func (a *App) initWorkspace(cmd *cobra.Command) error {
	// Create image if needed
	if !a.FS.Exists(a.Config.Image.Path) {
		a.Printer.Step("Creating sparse disk image...")
		if err := a.Exec.Run(cmd.Context(), "hdiutil", "create",
			"-size", a.Config.Image.Size,
			"-fs", "Case-sensitive APFS",
			"-volname", a.Config.Image.VolumeName,
			"-type", "SPARSE",
			a.Config.Image.Path,
		); err != nil {
			return fmt.Errorf("failed to create disk image: %w", err)
		}
		a.Printer.Success("Disk image created!")
	}

	// Mount if needed
	if !a.Context.IsMounted() {
		if err := a.mountVolume(cmd); err != nil {
			return err
		}
	}

	a.Printer.Success("Workspace initialized! Volume mounted at %s", a.Config.Image.MountPoint)
	return nil
}

func (a *App) mountVolume(cmd *cobra.Command) error {
	if a.Context.IsMounted() {
		a.Printer.Info("Volume already mounted at %s", a.Config.Image.MountPoint)
		return nil
	}

	a.Printer.Step("Mounting volume...")
	if err := a.Exec.Run(cmd.Context(), "hdiutil", "attach", a.Config.Image.Path); err != nil {
		return fmt.Errorf("failed to mount: %w", err)
	}
	a.Printer.Success("Volume mounted at %s", a.Config.Image.MountPoint)
	return nil
}

func (a *App) unmountVolume(cmd *cobra.Command) error {
	if !a.Context.IsMounted() {
		a.Printer.Info("Volume not mounted")
		return nil
	}

	a.Printer.Step("Unmounting volume...")
	if err := a.Exec.Run(cmd.Context(), "hdiutil", "detach", a.Config.Image.MountPoint); err != nil {
		return fmt.Errorf("failed to unmount: %w", err)
	}
	a.Printer.Success("Volume unmounted")
	return nil
}

func (a *App) cloneKernel(cmd *cobra.Command, url string) error {
	if err := a.Context.EnsureMounted(); err != nil {
		return err
	}

	if a.Context.KernelExists() {
		a.Printer.Info("Kernel source already exists at %s", a.Config.Paths.KernelDir)
		return nil
	}

	a.Printer.Step("Cloning kernel from %s...", url)
	a.Printer.Info("This may take a while...")

	if err := a.Exec.Run(cmd.Context(), "git", "clone", "--depth=1", url, a.Config.Paths.KernelDir); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}
	a.Printer.Success("Kernel cloned to %s", a.Config.Paths.KernelDir)
	return nil
}

func (a *App) buildConfigCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage elmos configuration",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			a.Printer.Print("Current Configuration:")
			a.Printer.Print("  Architecture:  %s", a.Config.Build.Arch)
			a.Printer.Print("  Jobs:          %d", a.Config.Build.Jobs)
			a.Printer.Print("  LLVM:          %v", a.Config.Build.LLVM)
			a.Printer.Print("  Memory:        %s", a.Config.QEMU.Memory)
			a.Printer.Print("  Project Root:  %s", a.Config.Paths.ProjectRoot)
			a.Printer.Print("  Volume:        %s", a.Config.Image.MountPoint)
		},
	}

	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long: `Set a configuration value. Valid keys:
  arch      - Target architecture (arm64, arm, riscv)
  jobs      - Number of parallel build jobs
  memory    - QEMU memory allocation`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			switch key {
			case "arch":
				if !config.IsValidArch(value) {
					return fmt.Errorf("invalid architecture: %s (valid: arm64, arm, riscv)", value)
				}
				a.Config.Build.Arch = value
			case "jobs":
				var jobs int
				if _, err := fmt.Sscanf(value, "%d", &jobs); err != nil {
					return fmt.Errorf("invalid jobs value: %s", value)
				}
				a.Config.Build.Jobs = jobs
			case "memory":
				a.Config.QEMU.Memory = value
			default:
				return fmt.Errorf("unknown key: %s", key)
			}

			// Save config
			configPath := filepath.Join(a.Config.Paths.ProjectRoot, "elmos.yaml")
			if err := a.Config.Save(configPath); err != nil {
				return err
			}
			a.Printer.Success("Set %s = %s", key, value)
			return nil
		},
	}

	initCfgCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file with defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			configPath := filepath.Join(cwd, "elmos.yaml")

			if a.FS.Exists(configPath) {
				a.Printer.Warn("Configuration file already exists: %s", configPath)
				return nil
			}

			// Create default config - paths will be auto-resolved
			cfg := &config.Config{
				Build: config.BuildConfig{
					Arch:         "arm64",
					LLVM:         true,
					CrossCompile: "llvm-",
				},
				Image: config.ImageConfig{
					Size:       "20G",
					VolumeName: "kernel-dev",
				},
				QEMU: config.QEMUConfig{
					Memory:  "2G",
					GDBPort: 1234,
					SSHPort: 2222,
				},
			}

			if err := cfg.Save(configPath); err != nil {
				return err
			}
			a.Printer.Success("Created configuration file: %s", configPath)
			return nil
		},
	}

	configCmd.AddCommand(showCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(initCfgCmd)
	return configCmd
}

func (a *App) buildDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment and dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Printer.Info("ELMOS Doctor - Environment Check")
			a.Printer.Print("")

			results, issues := a.HealthChecker.CheckAll(cmd.Context())

			currentSection := ""
			for _, r := range results {
				section := getSection(r.Name)
				if section != currentSection {
					a.Printer.Step("Checking %s...", section)
					currentSection = section
				}

				if r.Passed {
					a.Printer.Print("  ✓ %s", r.Name)
				} else if r.Required {
					a.Printer.Print("  ✗ %s (missing)", r.Name)
				} else {
					a.Printer.Print("  ○ %s - optional", r.Name)
				}
			}

			if a.AutoFixer.CanFixElfH() {
				a.Printer.Print("")
				a.Printer.Step("Downloading missing elf.h...")
				if err := a.AutoFixer.FixElfH(); err != nil {
					a.Printer.Error("Failed to download elf.h: %v", err)
				} else {
					a.Printer.Success("elf.h downloaded")
					issues--
				}
			}

			a.Printer.Print("")
			if issues == 0 {
				a.Printer.Success("All checks passed!")
			} else {
				a.Printer.Warn("Found %d issue(s)", issues)
			}
			return nil
		},
	}
}

func (a *App) buildBuildCommand() *cobra.Command {
	var jobs int
	cmd := &cobra.Command{
		Use:   "build [targets...]",
		Short: "Build the Linux kernel",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}

			targets := args
			if len(targets) == 0 {
				targets = a.KernelBuilder.GetDefaultTargets()
			}

			a.Printer.Step("Building kernel for %s...", a.Config.Build.Arch)
			if err := a.KernelBuilder.Build(cmd.Context(), builder.BuildOptions{Jobs: jobs, Targets: targets}); err != nil {
				return err
			}
			a.Printer.Success("Build complete!")
			return nil
		},
	}
	cmd.Flags().IntVarP(&jobs, "jobs", "j", 0, "Number of parallel build jobs")
	return cmd
}

func (a *App) buildKernelCommand() *cobra.Command {
	kernelCmd := &cobra.Command{Use: "kernel", Short: "Kernel configuration commands"}

	configCmd := &cobra.Command{
		Use:   "config [type]",
		Short: "Configure the kernel",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}
			configType := "defconfig"
			if len(args) > 0 {
				configType = args[0]
			}
			a.Printer.Step("Running kernel %s...", configType)
			if err := a.KernelBuilder.Configure(cmd.Context(), configType); err != nil {
				return err
			}
			a.Printer.Success("Kernel configured!")
			return nil
		},
	}

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean kernel build artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}
			a.Printer.Step("Cleaning...")
			if err := a.KernelBuilder.Clean(cmd.Context()); err != nil {
				return err
			}
			a.Printer.Success("Cleaned!")
			return nil
		},
	}

	kernelCmd.AddCommand(configCmd, cleanCmd)
	return kernelCmd
}

func (a *App) buildModuleCommand() *cobra.Command {
	modCmd := &cobra.Command{Use: "module", Short: "Manage kernel modules"}

	buildCmd := &cobra.Command{
		Use: "build [name]", Short: "Build kernel modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			a.Printer.Step("Building modules...")
			if err := a.ModuleBuilder.Build(cmd.Context(), name); err != nil {
				return err
			}
			a.Printer.Success("Modules built!")
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			mods, err := a.ModuleBuilder.GetModules("")
			if err != nil {
				return err
			}
			if len(mods) == 0 {
				a.Printer.Info("No modules found")
				return nil
			}
			a.Printer.Print("Modules:")
			for i, m := range mods {
				status := ""
				if m.Built {
					status = " (built)"
				}
				a.Printer.Print("  %d. %s%s", i+1, m.Name, status)
			}
			return nil
		},
	}

	newCmd := &cobra.Command{
		Use: "new [name]", Short: "Create new module", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.ModuleBuilder.CreateModule(args[0]); err != nil {
				return err
			}
			a.Printer.Success("Created module: %s", args[0])
			return nil
		},
	}

	modCmd.AddCommand(buildCmd, listCmd, newCmd)
	return modCmd
}

func (a *App) buildAppsCommand() *cobra.Command {
	appCmd := &cobra.Command{Use: "app", Short: "Manage userspace applications"}

	buildCmd := &cobra.Command{
		Use: "build [name]", Short: "Build apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			a.Printer.Step("Building apps...")
			if err := a.AppBuilder.Build(cmd.Context(), name); err != nil {
				return err
			}
			a.Printer.Success("Apps built!")
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			apps, err := a.AppBuilder.GetApps("")
			if err != nil {
				return err
			}
			if len(apps) == 0 {
				a.Printer.Info("No apps found")
				return nil
			}
			a.Printer.Print("Applications:")
			for i, app := range apps {
				status := ""
				if app.Built {
					status = " (built)"
				}
				a.Printer.Print("  %d. %s%s", i+1, app.Name, status)
			}
			return nil
		},
	}

	newCmd := &cobra.Command{
		Use: "new [name]", Short: "Create new app", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.AppBuilder.CreateApp(args[0]); err != nil {
				return err
			}
			a.Printer.Success("Created app: %s", args[0])
			return nil
		},
	}

	appCmd.AddCommand(buildCmd, listCmd, newCmd)
	return appCmd
}

func (a *App) buildQEMUCommand() *cobra.Command {
	qemuCmd := &cobra.Command{Use: "qemu", Short: "Run and debug kernel in QEMU"}
	var graphical bool

	runCmd := &cobra.Command{
		Use: "run", Short: "Run kernel",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}
			a.Printer.Step("Starting QEMU...")
			return a.QEMURunner.Run(cmd.Context(), emulator.RunOptions{Graphical: graphical})
		},
	}
	runCmd.Flags().BoolVarP(&graphical, "graphical", "g", false, "Graphical mode")

	debugCmd := &cobra.Command{
		Use: "debug", Short: "Debug kernel",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}
			a.Printer.Step("Starting QEMU in debug mode...")
			return a.QEMURunner.Debug(cmd.Context(), graphical)
		},
	}

	gdbCmd := &cobra.Command{
		Use: "gdb", Short: "Connect GDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.QEMURunner.ConnectGDB()
		},
	}

	qemuCmd.AddCommand(runCmd, debugCmd, gdbCmd)
	return qemuCmd
}

func (a *App) buildRootfsCommand() *cobra.Command {
	rootfsCmd := &cobra.Command{Use: "rootfs", Short: "Manage root filesystem"}
	var size string

	createCmd := &cobra.Command{
		Use: "create", Short: "Create rootfs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}
			a.Printer.Step("Creating rootfs...")
			if err := a.RootfsCreator.Create(cmd.Context(), rootfs.CreateOptions{Size: size}); err != nil {
				return err
			}
			a.Printer.Success("Rootfs created!")
			return nil
		},
	}
	createCmd.Flags().StringVarP(&size, "size", "s", "5G", "Disk size")

	rootfsCmd.AddCommand(createCmd)
	return rootfsCmd
}

func (a *App) buildPatchCommand() *cobra.Command {
	patchCmd := &cobra.Command{Use: "patch", Short: "Manage kernel patches"}

	applyCmd := &cobra.Command{
		Use: "apply [file]", Short: "Apply patch", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Context.EnsureMounted(); err != nil {
				return err
			}
			a.Printer.Step("Applying patch: %s", args[0])
			if err := a.PatchManager.Apply(cmd.Context(), args[0]); err != nil {
				return err
			}
			a.Printer.Success("Patch applied!")
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List patches",
		RunE: func(cmd *cobra.Command, args []string) error {
			patches, err := a.PatchManager.List()
			if err != nil {
				return err
			}
			if len(patches) == 0 {
				a.Printer.Info("No patches")
				return nil
			}
			a.Printer.Print("Patches:")
			for _, p := range patches {
				a.Printer.Print("  %s/%s", p.Version, p.Name)
			}
			return nil
		},
	}

	patchCmd.AddCommand(applyCmd, listCmd)
	return patchCmd
}

func getSection(name string) string {
	if name == "Homebrew" {
		return "Homebrew"
	}
	if len(name) > 5 && name[:5] == "Tap: " {
		return "Homebrew taps"
	}
	if len(name) > 9 && name[:9] == "Package: " {
		return "required packages"
	}
	if len(name) > 8 && name[:8] == "Header: " {
		return "custom headers"
	}
	if len(name) > 5 && name[:5] == "GDB: " {
		return "cross-debuggers"
	}
	if len(name) > 5 && name[:5] == "GCC: " {
		return "cross-compilers"
	}
	return "other"
}
