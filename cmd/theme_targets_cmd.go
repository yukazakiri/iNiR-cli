package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var themeListTargetsCmd = &cobra.Command{
	Use:   "list-targets",
	Short: "List built-in and auto-discovered theme targets",
	RunE:  runThemeListTargets,
}

func runThemeListTargets(cmd *cobra.Command, args []string) error {
	configPath := defaultConfigPath()
	if flagConfig != "" {
		configPath = flagConfig
	}

	externalTargets, err := discoverExternalTargets(configPath)
	if err != nil {
		return err
	}

	builtin := allRegisteredTargets()
	sort.Strings(builtin)

	fmt.Fprintln(cmd.OutOrStdout(), "Built-in targets:")
	for _, targetName := range builtin {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", targetName)
	}

	externalIDs := listExternalTargetIDs(externalTargets)
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "External targets (auto-discovered):")
	if len(externalIDs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  (none)")
		return nil
	}
	for _, targetID := range externalIDs {
		spec := externalTargets[targetID]
		if spec.Description != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s - %s\n", targetID, spec.Description)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", targetID)
	}

	return nil
}
