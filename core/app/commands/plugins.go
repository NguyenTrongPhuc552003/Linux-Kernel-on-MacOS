// Package commands provides individual CLI command builders for elmos.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// BuildPlugins creates the plugins command with subcommands for plugin management.
func BuildPlugins(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage elmos plugins",
		Long:  "List, inspect, and manage elmos plugins",
	}

	cmd.AddCommand(buildPluginsListCmd(ctx))
	cmd.AddCommand(buildPluginsInfoCmd(ctx))
	cmd.AddCommand(buildPluginsCheckCmd(ctx))

	return cmd
}

// buildPluginsListCmd lists all loaded plugins.
func buildPluginsListCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all loaded plugins",
		Long:  "Display all loaded plugins with their versions and status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ctx.PluginRegistry == nil {
				ctx.Printer.Info("No plugins loaded")
				return nil
			}

			plugins := ctx.PluginRegistry.ListPlugins()
			if len(plugins) == 0 {
				ctx.Printer.Info("No plugins loaded")
				return nil
			}

			ctx.Printer.Info("Loaded Plugins (%d):", len(plugins))
			for _, p := range plugins {
				ctx.Printer.Step("  • %s (v%s) - %s", p.Name(), p.Version(), p.Description())
			}

			return nil
		},
	}
}

// buildPluginsInfoCmd shows detailed information about a specific plugin.
func buildPluginsInfoCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "info <plugin-name>",
		Short: "Show detailed information about a plugin",
		Long:  "Display detailed information including hooks and configuration for a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			if ctx.PluginRegistry == nil {
				return fmt.Errorf("no plugins loaded")
			}

			plugin := ctx.PluginRegistry.GetPlugin(pluginName)
			if plugin == nil {
				return fmt.Errorf("plugin '%s' not found", pluginName)
			}

			ctx.Printer.Info("Plugin: %s (v%s)", plugin.Name(), plugin.Version())
			ctx.Printer.Info("Description: %s", plugin.Description())

			hooks := plugin.Hooks()
			if len(hooks) > 0 {
				ctx.Printer.Info("Registered Hooks (%d):", len(hooks))
				for _, h := range hooks {
					requiredStr := "optional"
					if h.Required {
						requiredStr = "required"
					}
					ctx.Printer.Step("  • %s (priority: %d, %s)", h.Event, h.Priority, requiredStr)
				}
			} else {
				ctx.Printer.Info("No hooks registered")
			}

			return nil
		},
	}
}

// buildPluginsCheckCmd validates plugin compatibility.
func buildPluginsCheckCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate plugin compatibility",
		Long:  "Validate all loaded plugins for compatibility and dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ctx.PluginRegistry == nil {
				ctx.Printer.Warn("No plugins loaded")
				return nil
			}

			plugins := ctx.PluginRegistry.ListPlugins()
			if len(plugins) == 0 {
				ctx.Printer.Info("No plugins to validate")
				return nil
			}

			ctx.Printer.Info("Validating %d plugin(s)...", len(plugins))

			allValid := true
			for _, p := range plugins {
				err := p.Validate()
				if err != nil {
					ctx.Printer.Warn("  ✗ %s: %v", p.Name(), err)
					allValid = false
				} else {
					ctx.Printer.Step("  ✓ %s: OK", p.Name())
				}
			}

			if !allValid {
				return fmt.Errorf("one or more plugins failed validation")
			}

			ctx.Printer.Info("All plugins validated successfully")
			return nil
		},
	}
}
