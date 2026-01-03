// Package ui provides console output helpers for elmos.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Tokyo Night color palette
var (
	headerColor  = lipgloss.Color("141") // Purple
	commandColor = lipgloss.Color("120") // Green
	flagColor    = lipgloss.Color("214") // Orange
	descColor    = lipgloss.Color("245") // Grey
	exampleColor = lipgloss.Color("51")  // Cyan
	sectionColor = lipgloss.Color("255") // White
	subtleColor  = lipgloss.Color("238") // Dark grey
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(headerColor)
	commandStyle = lipgloss.NewStyle().Foreground(commandColor)
	flagStyle    = lipgloss.NewStyle().Foreground(flagColor)
	descStyle    = lipgloss.NewStyle().Foreground(descColor)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(sectionColor).MarginTop(1)
	exampleStyle = lipgloss.NewStyle().Foreground(exampleColor)
	subtleStyle  = lipgloss.NewStyle().Foreground(subtleColor)
)

// Banner returns a styled ASCII art banner for ELMOS.
func Banner() string {
	banner := `
 ███████╗██╗     ███╗   ███╗ ██████╗ ███████╗
 ██╔════╝██║     ████╗ ████║██╔═══██╗██╔════╝
 █████╗  ██║     ██╔████╔██║██║   ██║███████╗
 ██╔══╝  ██║     ██║╚██╔╝██║██║   ██║╚════██║
 ███████╗███████╗██║ ╚═╝ ██║╚██████╔╝███████║
 ╚══════╝╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚══════╝`
	return headerStyle.Render(banner)
}

// SetCustomUsageFunc sets a custom usage function for a Cobra command.
func SetCustomUsageFunc(cmd *cobra.Command) {
	cmd.SetUsageFunc(customUsageFunc)
	cmd.SetHelpFunc(customHelpFunc)
}

func customHelpFunc(cmd *cobra.Command, args []string) {
	var out strings.Builder

	// Show banner only for root command
	if !cmd.HasParent() {
		out.WriteString(Banner())
		out.WriteString("\n\n")
	}

	// Title and description
	if cmd.Short != "" {
		out.WriteString(headerStyle.Render(cmd.Short))
		out.WriteString("\n")
	}
	if cmd.Long != "" {
		out.WriteString(subtleStyle.Render(cmd.Long))
		out.WriteString("\n")
	}

	// Usage
	out.WriteString("\n")
	out.WriteString(sectionStyle.Render("USAGE"))
	out.WriteString("\n")
	out.WriteString("  " + commandStyle.Render(cmd.UseLine()))
	out.WriteString("\n")

	// Commands
	if cmd.HasAvailableSubCommands() {
		out.WriteString("\n")
		out.WriteString(sectionStyle.Render("COMMANDS"))
		out.WriteString("\n")

		// Only group commands for root command
		if !cmd.HasParent() {
			groups := groupCommands(cmd.Commands())
			for _, group := range groups {
				if group.name != "" {
					out.WriteString("  " + subtleStyle.Render("─── "+group.name+" ───"))
					out.WriteString("\n")
				}
				for _, sub := range group.commands {
					if sub.IsAvailableCommand() {
						name := commandStyle.Render(fmt.Sprintf("%-12s", sub.Name()))
						desc := descStyle.Render(sub.Short)
						out.WriteString(fmt.Sprintf("  %s  %s\n", name, desc))
					}
				}
			}
		} else {
			// Simple list for subcommands
			for _, sub := range cmd.Commands() {
				if sub.IsAvailableCommand() {
					name := commandStyle.Render(fmt.Sprintf("%-12s", sub.Name()))
					desc := descStyle.Render(sub.Short)
					out.WriteString(fmt.Sprintf("  %s  %s\n", name, desc))
				}
			}
		}
	}

	// Flags
	if cmd.HasAvailableLocalFlags() || cmd.HasAvailablePersistentFlags() {
		out.WriteString("\n")
		out.WriteString(sectionStyle.Render("FLAGS"))
		out.WriteString("\n")

		printFlags := func(flags *pflag.FlagSet) {
			flags.VisitAll(func(f *pflag.Flag) {
				if f.Hidden {
					return
				}
				var name string
				if f.Shorthand != "" {
					name = flagStyle.Render(fmt.Sprintf("  -%s, --%s", f.Shorthand, f.Name))
				} else {
					name = flagStyle.Render(fmt.Sprintf("      --%s", f.Name))
				}
				desc := descStyle.Render(f.Usage)
				out.WriteString(fmt.Sprintf("%-30s  %s\n", name, desc))
			})
		}
		printFlags(cmd.LocalFlags())
	}

	// Examples
	if cmd.Example != "" {
		out.WriteString("\n")
		out.WriteString(sectionStyle.Render("EXAMPLES"))
		out.WriteString("\n")
		for _, line := range strings.Split(cmd.Example, "\n") {
			out.WriteString("  " + exampleStyle.Render(line) + "\n")
		}
	}

	// Footer
	if cmd.HasAvailableSubCommands() {
		out.WriteString("\n")
		out.WriteString(subtleStyle.Render(fmt.Sprintf("Use \"%s [command] --help\" for more information about a command.", cmd.CommandPath())))
		out.WriteString("\n")
	}

	fmt.Fprint(cmd.OutOrStdout(), out.String())
}

func customUsageFunc(cmd *cobra.Command) error {
	customHelpFunc(cmd, nil)
	return nil
}

// commandGroup represents a group of related commands.
type commandGroup struct {
	name     string
	commands []*cobra.Command
}

// groupCommands organizes commands into logical groups.
func groupCommands(cmds []*cobra.Command) []commandGroup {
	core := []*cobra.Command{}
	build := []*cobra.Command{}
	runtime := []*cobra.Command{}
	config := []*cobra.Command{}
	other := []*cobra.Command{}

	for _, cmd := range cmds {
		if !cmd.IsAvailableCommand() {
			continue
		}
		switch cmd.Name() {
		case "init", "exit", "doctor", "version", "tui", "status", "arch":
			core = append(core, cmd)
		case "build", "kernel", "module", "app", "rootfs", "patch":
			build = append(build, cmd)
		case "qemu", "gdb":
			runtime = append(runtime, cmd)
		default:
			other = append(other, cmd)
		}
	}

	groups := []commandGroup{}
	if len(core) > 0 {
		groups = append(groups, commandGroup{name: "Core", commands: core})
	}
	if len(build) > 0 {
		groups = append(groups, commandGroup{name: "Build", commands: build})
	}
	if len(runtime) > 0 {
		groups = append(groups, commandGroup{name: "Runtime", commands: runtime})
	}
	if len(config) > 0 {
		groups = append(groups, commandGroup{name: "Config", commands: config})
	}
	if len(other) > 0 {
		groups = append(groups, commandGroup{name: "", commands: other})
	}

	return groups
}
