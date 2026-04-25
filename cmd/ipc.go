// File: ipc.go
//
// IPC command wiring for iNiR Quickshell shell communication.
//
// Two invocation modes:
//   - inir-cli ipc <target> <function> [args]  — Raw passthrough (no validation)
//   - inir-cli <target> <function> [args]      — Validated, with kebab-case aliases
//
// The raw "ipc" subcommand skips normalization and validation so future
// shell functions aren't blocked by an outdated CLI registry.
// Target subcommands (e.g. "overview toggle") validate the function name
// against the registry and provide per-target help output.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newIPCRootCommand())
	for _, meta := range ipcTargets {
		rootCmd.AddCommand(newIPCTargetCommand(meta))
	}
}

func newIPCRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "ipc [-c PATH] <target> <function> [args...]",
		Short:              "Call a raw iNiR Quickshell IPC target",
		DisableFlagParsing: true,
		RunE:               runRawIPCCommand,
	}
}

func runRawIPCCommand(cmd *cobra.Command, args []string) error {
	parsed, err := parseIPCPrefixArgs(args)
	if err != nil {
		return err
	}
	if parsed.Help {
		return cmd.Help()
	}
	if len(parsed.Rest) < 2 {
		return fmt.Errorf("usage: inir-cli ipc <target> <function> [args...]")
	}
	return runIPCCommand(parsed.Config, parsed.Rest)
}

func newIPCTargetCommand(meta ipcTarget) *cobra.Command {
	return &cobra.Command{
		Use:                meta.Name + " [-c PATH] <function> [args...]",
		Aliases:            ipcAliasesForTarget(meta.Name),
		Short:              firstSentence(meta.Description),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIPCTargetCommand(cmd, meta, args)
		},
	}
}

func runIPCTargetCommand(cmd *cobra.Command, meta ipcTarget, args []string) error {
	parsed, err := parseIPCPrefixArgs(args)
	if err != nil {
		return err
	}
	if parsed.Help {
		printIPCTargetHelp(cmd.OutOrStdout(), meta)
		return nil
	}
	if len(parsed.Rest) == 0 {
		return runDefaultIPCTargetAction(cmd, meta, parsed.Config)
	}
	if err := validateIPCFunction(meta, parsed.Rest[0]); err != nil {
		return err
	}
	return runIPCCommand(parsed.Config, append([]string{meta.Name}, parsed.Rest...))
}

func runDefaultIPCTargetAction(cmd *cobra.Command, meta ipcTarget, configDir string) error {
	if meta.Name == "settings" {
		return runIPCCommand(configDir, []string{meta.Name, "open"})
	}
	printIPCTargetHelp(cmd.ErrOrStderr(), meta)
	return fmt.Errorf("missing function for IPC target %q", meta.Name)
}
