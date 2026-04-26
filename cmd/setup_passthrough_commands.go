// File: setup_passthrough_commands.go
//
// Upstream maintenance-command passthroughs.
//
// These commands mirror upstream launcher behavior by delegating to the
// upstream `setup` script after stripping a leading compatibility `-c/--config`
// flag pair (if present).
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type setupPassthroughSpec struct {
	Name        string
	Description string
	Usage       string
}

var setupPassthroughCommands = []setupPassthroughSpec{
	{Name: "doctor", Description: "Run upstream maintenance doctor flow", Usage: "inir-cli doctor [-c PATH] [setup-args...]"},
	{Name: "status", Description: "Run upstream maintenance status flow", Usage: "inir-cli status [-c PATH] [setup-args...]"},
	{Name: "migrate", Description: "Run upstream maintenance migration flow", Usage: "inir-cli migrate [-c PATH] [setup-args...]"},
	{Name: "reinstall", Description: "Run upstream maintenance reinstall flow", Usage: "inir-cli reinstall [-c PATH] [setup-args...]"},
	{Name: "rollback", Description: "Run upstream maintenance rollback flow", Usage: "inir-cli rollback [-c PATH] [setup-args...]"},
	{Name: "my-changes", Description: "Run upstream maintenance user-modifications flow", Usage: "inir-cli my-changes [-c PATH] [setup-args...]"},
	{Name: "uninstall", Description: "Run upstream maintenance uninstall flow", Usage: "inir-cli uninstall [-c PATH] [setup-args...]"},
	{Name: "config", Description: "Run upstream maintenance config flow", Usage: "inir-cli config [-c PATH] [setup-args...]"},
	{Name: "info", Description: "Run upstream maintenance info flow", Usage: "inir-cli info [-c PATH] [setup-args...]"},
	{Name: "backup", Description: "Run upstream maintenance backup flow", Usage: "inir-cli backup [-c PATH] [setup-args...]"},
	{Name: "logs", Description: "Run upstream maintenance logs flow", Usage: "inir-cli logs [-c PATH] [setup-args...]"},
}

func init() {
	for _, spec := range setupPassthroughCommands {
		rootCmd.AddCommand(newSetupPassthroughCommand(spec))
	}
	rootCmd.AddCommand(newSetupEntrypointCommand())
}

func newSetupPassthroughCommand(spec setupPassthroughSpec) *cobra.Command {
	return &cobra.Command{
		Use:                fmt.Sprintf("%s [-c PATH] [setup-args...]", spec.Name),
		Short:              spec.Description,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetupMaintenanceCommand(cmd, args, spec.Name, spec.Usage)
		},
	}
}

func newSetupEntrypointCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "setup [-c PATH] [setup-args...]",
		Short:              "Run upstream setup entrypoint",
		DisableFlagParsing: true,
		RunE:               runSetupEntrypointCommand,
	}
}

func runSetupMaintenanceCommand(cmd *cobra.Command, args []string, maintenanceCommand, usage string) error {
	rest, err := stripLeadingConfigCompatArgs(args)
	if err != nil {
		return err
	}

	if len(rest) > 0 && isHelpFlag(rest[0]) {
		fmt.Fprintln(cmd.OutOrStdout(), "Usage:", usage)
		return nil
	}

	setupDir, err := setupDirResolver()
	if err != nil {
		return err
	}

	setupArgs := append([]string{maintenanceCommand}, rest...)
	return setupCommandRunner(setupDir, setupArgs)
}

func runSetupEntrypointCommand(cmd *cobra.Command, args []string) error {
	rest, err := stripLeadingConfigCompatArgs(args)
	if err != nil {
		return err
	}

	if len(rest) > 0 && isHelpFlag(rest[0]) {
		fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli setup [-c PATH] [setup-args...]")
		return nil
	}

	setupDir, err := setupDirResolver()
	if err != nil {
		return err
	}

	return setupCommandRunner(setupDir, rest)
}

func stripLeadingConfigCompatArgs(args []string) ([]string, error) {
	index := 0
	for index < len(args) {
		switch args[index] {
		case "-c", "--config":
			if index+1 >= len(args) {
				return nil, fmt.Errorf("missing path after %s", args[index])
			}
			index += 2
		default:
			return args[index:], nil
		}
	}
	return []string{}, nil
}

func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help"
}
