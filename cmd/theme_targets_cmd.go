// File: theme_targets_cmd.go
//
// Cobra commands for target management:
//   - theme list-targets:     Lists all built-in and auto-discovered external targets
//   - theme scaffold-target:  Creates a JSON spec file for a new external target
//
// Scaffold writes a validated JSON spec to the primary target discovery directory,
// enabling one-file community onboarding for new theme targets.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagScaffoldCommand     string
	flagScaffoldDescription string
	flagScaffoldArgs        []string
	flagScaffoldInputs      []string
	flagScaffoldEnv         []string
	flagScaffoldEnabled     bool
	flagScaffoldDir         string
	flagScaffoldForce       bool
)

var themeListTargetsCmd = &cobra.Command{
	Use:   "list-targets",
	Short: "List built-in and auto-discovered theme targets",
	RunE:  runThemeListTargets,
}

var themeScaffoldTargetCmd = &cobra.Command{
	Use:   "scaffold-target <id>",
	Short: "Create an external theme target JSON spec",
	Args:  cobra.ExactArgs(1),
	RunE:  runThemeScaffoldTarget,
}

func init() {
	themeScaffoldTargetCmd.Flags().StringVar(&flagScaffoldCommand, "command", "", "Executable command path for this target (required)")
	themeScaffoldTargetCmd.Flags().StringVar(&flagScaffoldDescription, "description", "", "Human-readable description")
	themeScaffoldTargetCmd.Flags().StringArrayVar(&flagScaffoldArgs, "arg", nil, "Command argument (repeatable)")
	themeScaffoldTargetCmd.Flags().StringArrayVar(&flagScaffoldInputs, "input", nil, "Declared input contract file (repeatable)")
	themeScaffoldTargetCmd.Flags().StringArrayVar(&flagScaffoldEnv, "env", nil, "Environment variable KEY=VALUE (repeatable)")
	themeScaffoldTargetCmd.Flags().BoolVar(&flagScaffoldEnabled, "enabled", true, "Whether this external target is enabled")
	themeScaffoldTargetCmd.Flags().StringVar(&flagScaffoldDir, "dir", "", "Target directory override (defaults to auto-discovery primary location)")
	themeScaffoldTargetCmd.Flags().BoolVar(&flagScaffoldForce, "force", false, "Overwrite existing target file")
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

func runThemeScaffoldTarget(cmd *cobra.Command, args []string) error {
	targetID := strings.TrimSpace(args[0])
	if !validTargetIDPattern.MatchString(targetID) {
		return fmt.Errorf("invalid target id %q (expected [a-z0-9][a-z0-9-]*)", targetID)
	}
	if strings.TrimSpace(flagScaffoldCommand) == "" {
		return fmt.Errorf("--command is required")
	}

	configPath := defaultConfigPath()
	if flagConfig != "" {
		configPath = flagConfig
	}

	targetDir := resolveScaffoldTargetDir(configPath)
	if flagScaffoldDir != "" {
		targetDir = flagScaffoldDir
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target dir %s: %w", targetDir, err)
	}

	targetPath := filepath.Join(targetDir, targetID+".json")
	if !flagScaffoldForce {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("target spec already exists: %s (use --force to overwrite)", targetPath)
		}
	}

	spec := externalThemeTarget{
		ID:          targetID,
		Type:        "command",
		Description: strings.TrimSpace(flagScaffoldDescription),
		Command:     strings.TrimSpace(flagScaffoldCommand),
		Args:        append([]string{}, flagScaffoldArgs...),
		Inputs:      append([]string{}, flagScaffoldInputs...),
	}
	if len(flagScaffoldEnv) > 0 {
		envMap, err := parseScaffoldEnv(flagScaffoldEnv)
		if err != nil {
			return err
		}
		spec.Env = envMap
	}
	if !flagScaffoldEnabled {
		enabled := false
		spec.Enabled = &enabled
	}

	encoded, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("encode target spec: %w", err)
	}
	if err := os.WriteFile(targetPath, append(encoded, '\n'), 0644); err != nil {
		return fmt.Errorf("write target spec %s: %w", targetPath, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created target spec: %s\n", targetPath)
	return nil
}

func resolveScaffoldTargetDir(configPath string) string {
	dirs := resolveExternalTargetDirs(configPath)
	if len(dirs) > 0 {
		return dirs[0]
	}
	return filepath.Join(filepath.Dir(configPath), "targets")
}

func parseScaffoldEnv(entries []string) (map[string]string, error) {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --env entry %q (expected KEY=VALUE)", entry)
		}
		result[key] = value
	}
	return result, nil
}
