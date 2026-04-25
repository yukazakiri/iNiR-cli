// File: ipc_runtime.go
//
// Runtime directory resolution and Quickshell binary execution for IPC calls.
//
// Resolution order for the iNiR shell payload:
//   1. Explicit -c/--config path
//   2. INIR_RUNTIME_DIR environment variable
//   3. XDG_CONFIG_HOME/quickshell/inir
//   4. INIR_SYSTEM_RUNTIME_DIR (default: /usr/local/share/quickshell/inir)
//   5. INIR_FALLBACK_SYSTEM_RUNTIME_DIR (default: /usr/share/quickshell/inir)
//
// The qs binary path is resolved via INIR_QS_BIN env var, falling back to /usr/bin/qs.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var ipcCommandRunner = runQSIPCCommand

func runIPCCommand(configDir string, callArgs []string) error {
	runtimeDir, err := resolveIPCRuntimeDir(configDir)
	if err != nil {
		return err
	}
	return ipcCommandRunner(runtimeDir, callArgs)
}

func resolveIPCRuntimeDir(configDir string) (string, error) {
	if configDir != "" {
		return validateIPCRuntimeDir(configDir)
	}
	if runtimeDir := os.Getenv("INIR_RUNTIME_DIR"); runtimeDir != "" {
		return validateIPCRuntimeDir(runtimeDir)
	}
	for _, candidate := range defaultIPCRuntimeCandidates() {
		if hasShellPayload(candidate) {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("could not find an iNiR shell payload; pass -c PATH or set INIR_RUNTIME_DIR")
}

func runQSIPCCommand(configDir string, callArgs []string) error {
	args := append([]string{"-p", configDir, "ipc", "call"}, callArgs...)
	command := exec.Command(resolveQSBinary(), args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func resolveQSBinary() string {
	if binary := os.Getenv("INIR_QS_BIN"); binary != "" {
		return binary
	}
	return "/usr/bin/qs"
}

func validateIPCRuntimeDir(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if !hasShellPayload(absDir) {
		return "", fmt.Errorf("invalid iNiR runtime path %q: shell.qml not found", dir)
	}
	return absDir, nil
}

func defaultIPCRuntimeCandidates() []string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return []string{
		filepath.Join(configHome, "quickshell", "inir"),
		envOrDefault("INIR_SYSTEM_RUNTIME_DIR", "/usr/local/share/quickshell/inir"),
		envOrDefault("INIR_FALLBACK_SYSTEM_RUNTIME_DIR", "/usr/share/quickshell/inir"),
	}
}

func hasShellPayload(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "shell.qml"))
	return err == nil && !info.IsDir()
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
