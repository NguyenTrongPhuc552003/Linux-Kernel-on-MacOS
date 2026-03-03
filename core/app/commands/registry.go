// Package commands provides individual CLI command builders for elmos.
// Each command group is in its own file for maintainability.
package commands

import (
	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/core/config"
	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/builder"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/doctor"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/emulator"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/patch"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/rootfs"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/toolchain"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/filesystem"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
	"github.com/NguyenTrongPhuc552003/elmos/core/ui"
)

// Context provides dependencies for command builders.
// This avoids circular imports by passing only what commands need.
type Context struct {
	Exec             executor.Executor
	FS               filesystem.FileSystem
	Config           *config.Config
	AppContext       *elcontext.Context
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

	// Flags that can be modified
	Verbose    *bool
	ConfigFile *string

	// ========== NEW v2.0 PLUGIN FIELDS ==========

	// HookExecutor manages build lifecycle hooks
	HookExecutor *plugin.HookExecutor

	// PluginRegistry manages loaded plugins
	PluginRegistry *plugin.Registry
}

// Register adds all subcommands to the root command.
func Register(ctx *Context, rootCmd *cobra.Command) {
	rootCmd.AddCommand(BuildVersion(ctx))
	rootCmd.AddCommand(BuildTUI(ctx))
	rootCmd.AddCommand(BuildInit(ctx))
	rootCmd.AddCommand(BuildExit(ctx))
	rootCmd.AddCommand(BuildArch(ctx))
	rootCmd.AddCommand(BuildDoctor(ctx))
	rootCmd.AddCommand(BuildKernel(ctx))
	rootCmd.AddCommand(BuildModule(ctx))
	rootCmd.AddCommand(BuildApps(ctx))
	rootCmd.AddCommand(BuildQEMU(ctx))
	rootCmd.AddCommand(BuildGDB(ctx))
	rootCmd.AddCommand(BuildStatus(ctx))
	rootCmd.AddCommand(BuildRootfs(ctx))
	rootCmd.AddCommand(BuildPatch(ctx))
	rootCmd.AddCommand(BuildToolchains(ctx))
	// NEW v2.0 commands
	rootCmd.AddCommand(BuildPlugins(ctx))
	rootCmd.AddCommand(BuildBootloader(ctx))
}
