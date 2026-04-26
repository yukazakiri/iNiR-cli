// File: version_command.go
//
// Upstream-compatible version command.
//
// Mirrors `inir version` behavior:
//   - Supports `--json`
//   - Prints a human-readable version summary when metadata is available
//   - Falls back to VERSION file from resolved setup directory
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newVersionCommand())
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "version [--json]",
		Short:              "Show iNiR runtime version metadata",
		DisableFlagParsing: true,
		RunE:               runVersionCommand,
	}
}

func runVersionCommand(cmd *cobra.Command, args []string) error {
	jsonMode := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonMode = true
		case "-h", "--help":
			fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli version [--json]")
			return nil
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	runtimePath := detectVersionRuntimePath()
	localVersionFile := filepath.Join(configHome(), "inir", "version.json")
	runtimeVersionFile := ""
	if runtimePath != "" {
		runtimeVersionFile = filepath.Join(runtimePath, "version.json")
	}

	versionFile := selectVersionFile(runtimeVersionFile, localVersionFile)
	if versionFile != "" {
		data, err := os.ReadFile(versionFile)
		if err == nil {
			if jsonMode {
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			if err := printHumanVersion(cmd, data, runtimePath); err == nil {
				return nil
			}
		}
	}

	setupDir, err := setupDirResolver()
	if err != nil {
		if jsonMode {
			fallback := map[string]string{"version": "unknown", "runtime": runtimePath}
			encoded, _ := json.MarshalIndent(fallback, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "iNiR %s\n", "unknown")
		if runtimePath != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "runtime: %s\n", runtimePath)
		}
		return nil
	}

	version := readVersionFromSetup(setupDir)
	if jsonMode {
		fallback := map[string]string{"version": version, "runtime": nonEmpty(runtimePath, setupDir)}
		encoded, _ := json.MarshalIndent(fallback, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "iNiR %s\n", version)
	fmt.Fprintf(cmd.OutOrStdout(), "runtime: %s\n", nonEmpty(runtimePath, setupDir))
	return nil
}

func detectVersionRuntimePath() string {
	if forced := os.Getenv("INIR_RUNTIME_DIR"); forced != "" {
		if hasShellPayload(forced) {
			abs, _ := filepath.Abs(forced)
			return abs
		}
	}

	for _, candidate := range defaultIPCRuntimeCandidates() {
		if hasShellPayload(candidate) {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}

	return ""
}

func selectVersionFile(runtimeVersionFile, localVersionFile string) string {
	if runtimeVersionFile != "" && os.Getenv("INIR_RUNTIME_DIR") != "" && fileExists(runtimeVersionFile) {
		return runtimeVersionFile
	}
	if fileExists(localVersionFile) {
		return localVersionFile
	}
	if runtimeVersionFile != "" && fileExists(runtimeVersionFile) {
		return runtimeVersionFile
	}
	return ""
}

func printHumanVersion(cmd *cobra.Command, versionJSON []byte, runtimePath string) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(versionJSON, &payload); err != nil {
		return err
	}

	version := readStringField(payload, "version", "unknown")
	commit := readStringField(payload, "commit", "unknown")
	installMode := readStringField(payload, "installMode", readStringField(payload, "install_mode", "unknown"))
	updateStrategy := readStringField(payload, "updateStrategy", readStringField(payload, "update_strategy", "unknown"))
	source := readStringField(payload, "source", "unknown")

	fmt.Fprintf(cmd.OutOrStdout(), "iNiR %s\n", version)
	fmt.Fprintf(cmd.OutOrStdout(), "commit: %s\n", commit)
	fmt.Fprintf(cmd.OutOrStdout(), "install mode: %s\n", installMode)
	fmt.Fprintf(cmd.OutOrStdout(), "update strategy: %s\n", updateStrategy)
	fmt.Fprintf(cmd.OutOrStdout(), "source: %s\n", source)
	fmt.Fprintf(cmd.OutOrStdout(), "runtime: %s\n", runtimePath)

	repoDir := readStringField(payload, "repoPath", "")
	if repoDir == "" {
		repoDir = runtimePath
	}
	if repoDir != "" && fileExists(filepath.Join(repoDir, ".git")) {
		if branch := gitBranch(repoDir); branch != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "branch: %s\n", branch)
			if branch != "main" && branch != "master" {
				fmt.Fprintln(cmd.OutOrStdout(), "note: non-release branch")
			}
		}
	}

	return nil
}

func gitBranch(repoDir string) string {
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readVersionFromSetup(setupDir string) string {
	versionPath := filepath.Join(setupDir, "VERSION")
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return "unknown"
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "unknown"
	}
	return value
}

func configHome() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		return configHome
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func readStringField(payload map[string]interface{}, key, fallback string) string {
	raw, ok := payload[key]
	if !ok {
		return fallback
	}
	value, ok := raw.(string)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func nonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
