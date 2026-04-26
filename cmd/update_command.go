// File: update_command.go
//
// Upstream-compatible `update` maintenance command.
//
// This command mirrors `inir update` behavior by delegating to the upstream
// `setup update` flow when an upstream install/repo is present.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	setupDirResolver    = resolveSetupDir
	setupCommandRunner  = runSetupCommand
	setupLauncherFinder = resolveExecutableDir
)

func init() {
	rootCmd.AddCommand(newUpdateCommand())
}

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "update [-c PATH] [-y|--yes] [-q|--quiet] [--local]",
		Short:              "Run upstream maintenance update flow",
		DisableFlagParsing: true,
		RunE:               runUpdateCommand,
	}
}

func runUpdateCommand(cmd *cobra.Command, args []string) error {
	forwarded, err := stripLeadingConfigCompatArgs(args)
	if err != nil {
		return err
	}

	if len(forwarded) > 0 && isHelpFlag(forwarded[0]) {
		fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli update [-c PATH] [-y|--yes] [-q|--quiet] [--local]")
		return nil
	}

	setupDir, err := setupDirResolver()
	if err != nil {
		return err
	}

	setupArgs := append([]string{"update"}, forwarded...)
	return setupCommandRunner(setupDir, setupArgs)
}

func runSetupCommand(setupDir string, args []string) error {
	setupScript := filepath.Join(setupDir, "setup")
	execCmd := exec.Command(setupScript, args...)
	execCmd.Dir = setupDir
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func resolveSetupDir() (string, error) {
	if fromEnv := os.Getenv("INIR_SETUP_DIR"); fromEnv != "" {
		if isValidSetupDir(fromEnv) {
			return fromEnv, nil
		}
	}

	if fromLauncher := setupLauncherFinder(); fromLauncher != "" && isValidSetupDir(fromLauncher) {
		return fromLauncher, nil
	}

	if fromVersion := resolveSetupDirFromVersionFile(); fromVersion != "" && isValidSetupDir(fromVersion) {
		return fromVersion, nil
	}

	if cwd, err := os.Getwd(); err == nil && isValidSetupDir(cwd) {
		return cwd, nil
	}

	return "", fmt.Errorf("unable to locate upstream setup script (set INIR_SETUP_DIR or install upstream iNiR)")
}

func resolveExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exePath)
}

func resolveSetupDirFromVersionFile() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	versionPath := filepath.Join(configHome, "inir", "version.json")

	data, err := os.ReadFile(versionPath)
	if err != nil {
		return ""
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}

	for _, key := range []string{"repoPath", "repo_path"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		pathValue, ok := raw.(string)
		if !ok || pathValue == "" {
			continue
		}
		if len(pathValue) > 1 && pathValue[0] == '~' && pathValue[1] == '/' {
			pathValue = filepath.Join(os.Getenv("HOME"), pathValue[2:])
		}
		return pathValue
	}

	return ""
}

func isValidSetupDir(path string) bool {
	setupScript := filepath.Join(path, "setup")
	shellQML := filepath.Join(path, "shell.qml")
	if info, err := os.Stat(setupScript); err == nil {
		if info.Mode().Perm()&0111 == 0 {
			return false
		}
	} else {
		return false
	}

	if _, err := os.Stat(shellQML); err != nil {
		return false
	}

	return true
}
