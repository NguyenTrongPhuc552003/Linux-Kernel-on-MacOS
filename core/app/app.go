// Package app provides the CLI application layer for elmos.
package app

import (
	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/core/app/commands"
	"github.com/NguyenTrongPhuc552003/elmos/core/app/version"
	"github.com/NguyenTrongPhuc552003/elmos/core/config"
	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/builder"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/doctor"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/emulator"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/patch"
	_ "github.com/NguyenTrongPhuc552003/elmos/core/domain/plugin/builtin"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/rootfs"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/toolchain"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
	"github.com/NguyenTrongPhuc552003/elmos/core/ui"
)

// App holds all the application dependencies.
type App struct {
	Exec             executor.Executor
	FS               filesystem.FileSystem
	Config           *config.Config
	Context          *elcontext.Context
	KernelBuilder    *builder.KernelBuilder
	ModuleBuilder    *builder.ModuleBuilder
	AppBuilder       *builder.AppBuilder
	QEMURunner       *emulator.QEMURunner
	HealthChecker    *doctor.HealthChecker
	AutoFixer        *doctor.AutoFixer
	RootfsCreator    *rootfs.Creator
	PatchManager     *patch.Manager
	ToolchainManager *toolchain.Manager
	Printer          *ui.Printer
	Verbose          bool
	ConfigFile       string

	// ========== NEW v2.0 FIELDS ==========

	// HookExecutor manages build lifecycle hooks from plugins
	HookExecutor *plugin.HookExecutor

	// PluginRegistry manages loaded plugins
	PluginRegistry *plugin.Registry
}

// New creates a new App with all dependencies wired up.
func New(exec executor.Executor, fs filesystem.FileSystem, cfg *config.Config) *App {
	ctx := elcontext.New(cfg, exec, fs)
	printer := ui.NewPrinter()
	tm := toolchain.NewManager(exec, fs, cfg, printer)

	// Initialize plugin system
	hookExecutor := plugin.NewHookExecutor(printer, cfg.Build.Verbose)
	pluginRegistry := plugin.NewRegistry(hookExecutor, printer, cfg.Build.Verbose)

	// Load builtin plugins
	// Convert plugin entries to map[string]interface{} for flexibility
	pluginConfigMap := make(map[string]interface{})
	for name, entry := range cfg.Plugins.Plugins {
		pluginConfigMap[name] = entry
	}
	if err := pluginRegistry.LoadBuiltins(ctx, pluginConfigMap); err != nil {
		// Log error but don't fail - plugins are optional
		printer.Warn("Failed to load some plugins: %v", err)
	}

	app := &App{
		Exec:             exec,
		FS:               fs,
		Config:           cfg,
		Context:          ctx,
		KernelBuilder:    builder.NewKernelBuilder(exec, fs, cfg, ctx, tm),
		ModuleBuilder:    builder.NewModuleBuilder(exec, fs, cfg, ctx, tm),
		AppBuilder:       builder.NewAppBuilder(exec, fs, cfg, ctx, tm),
		QEMURunner:       emulator.NewQEMURunner(exec, fs, cfg, ctx),
		HealthChecker:    doctor.NewHealthChecker(exec, fs, cfg, tm),
		AutoFixer:        doctor.NewAutoFixer(fs, cfg),
		RootfsCreator:    rootfs.NewCreator(exec, fs, cfg),
		PatchManager:     patch.NewManager(exec, fs, cfg),
		ToolchainManager: tm,
		Printer:          printer,
		HookExecutor:     hookExecutor,
		PluginRegistry:   pluginRegistry,
	}

	return app
}

// BuildRootCommand builds the root cobra command with all subcommands.
func (a *App) BuildRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "elmos",
		Short: "Embedded Linux SDK - Native kernel build tools",
		Long: `ELMOS provides native Linux kernel build tools for embedded development.

Common workflow:
  elmos init              # Initialize workspace
  elmos doctor            # Check dependencies
  elmos kernel config     # Configure kernel
  elmos build             # Build kernel
  elmos qemu run          # Test in QEMU
  elmos tui               # Launch interactive TUI`,
		Version: version.Get().String(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "tui" || cmd.Name() == "init" {
				return nil
			}
			// Reload config if custom path is provided
			if a.ConfigFile != "" {
				newCfg, err := config.Load(a.ConfigFile)
				if err != nil {
					return err
				}
				// update the struct contents so pointers passed to builders remain valid
				*a.Config = *newCfg
			}
			a.Context.Verbose = a.Verbose
			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&a.Verbose, "verbose", "e", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&a.ConfigFile, "config", "c", "", "config file path (auto-detected as <workspace>/<workspace>.yaml if omitted)")

	// Create command context and register all commands
	cmdCtx := &commands.Context{
		Exec:             a.Exec,
		FS:               a.FS,
		Config:           a.Config,
		AppContext:       a.Context,
		KernelBuilder:    a.KernelBuilder,
		ModuleBuilder:    a.ModuleBuilder,
		AppBuilder:       a.AppBuilder,
		QEMURunner:       a.QEMURunner,
		HealthChecker:    a.HealthChecker,
		AutoFixer:        a.AutoFixer,
		RootfsCreator:    a.RootfsCreator,
		PatchManager:     a.PatchManager,
		ToolchainManager: a.ToolchainManager,
		Printer:          a.Printer,
		Verbose:          &a.Verbose,
		ConfigFile:       &a.ConfigFile,
		// NEW v2.0 plugin fields
		HookExecutor:   a.HookExecutor,
		PluginRegistry: a.PluginRegistry,
	}

	commands.Register(cmdCtx, rootCmd)

	// Apply custom styled help output
	ui.SetCustomUsageFunc(rootCmd)

	return rootCmd
}
