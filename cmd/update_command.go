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
	cliUpdateRunner     = runCLIUpdate
)

const inirCLIModule = "github.com/yukazakiri/inir-cli"

func init() {
	rootCmd.AddCommand(newUpdateCommand())
}

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "update [-c PATH] [--cli [--version VERSION]] [-y|--yes] [-q|--quiet] [--local]",
		Short:              "Update upstream iNiR or inir-cli itself",
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
		fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli update [-c PATH] [--cli [--version VERSION]] [-y|--yes] [-q|--quiet] [--local]")
		fmt.Fprintln(cmd.OutOrStdout(), "  default: forwards to upstream setup update")
		fmt.Fprintln(cmd.OutOrStdout(), "  --cli:   updates inir-cli itself via `go install github.com/yukazakiri/inir-cli@latest`")
		return nil
	}

	updateCLI, version, setupForwarded, err := parseUpdateArgs(forwarded)
	if err != nil {
		return err
	}

	if updateCLI {
		return cliUpdateRunner(cmd, version)
	}

	setupDir, err := setupDirResolver()
	if err != nil {
		return err
	}

	setupArgs := append([]string{"update"}, setupForwarded...)
	return setupCommandRunner(setupDir, setupArgs)
}

func parseUpdateArgs(args []string) (bool, string, []string, error) {
	updateCLI := false
	version := ""
	setupForwarded := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--cli", "--self":
			updateCLI = true
		case "--upstream":
			updateCLI = false
		case "--version":
			if i+1 >= len(args) {
				return false, "", nil, fmt.Errorf("missing version after --version")
			}
			version = args[i+1]
			i++
		default:
			setupForwarded = append(setupForwarded, args[i])
		}
	}

	if version != "" && !updateCLI {
		return false, "", nil, fmt.Errorf("--version is only valid with --cli")
	}

	if updateCLI && len(setupForwarded) > 0 {
		return false, "", nil, fmt.Errorf("unsupported args for --cli: %v", setupForwarded)
	}

	return updateCLI, version, setupForwarded, nil
}

func runCLIUpdate(cmd *cobra.Command, version string) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go is required for --cli update")
	}

	if version == "" {
		version = "latest"
	}

	target := fmt.Sprintf("%s@%s", inirCLIModule, version)
	fmt.Fprintf(cmd.OutOrStdout(), "Updating inir-cli: go install %s\n", target)

	installCmd := exec.Command("go", "install", target)
	installCmd.Stdin = os.Stdin
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("inir-cli update failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "inir-cli update complete")
	return nil
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
