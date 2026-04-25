// File: ipc_runtime.go
//
// Runtime directory resolution, Quickshell binary execution, and lifecycle
// management for IPC calls.
//
// Resolution order for the iNiR shell payload:
//   1. Explicit -c/--config path
//   2. INIR_RUNTIME_DIR environment variable
//   3. XDG_CONFIG_HOME/quickshell/inir
//   4. INIR_SYSTEM_RUNTIME_DIR (default: /usr/local/share/quickshell/inir)
//   5. INIR_FALLBACK_SYSTEM_RUNTIME_DIR (default: /usr/share/quickshell/inir)
//
// The qs binary path is resolved via INIR_QS_BIN env var, falling back to /usr/bin/qs.
//
// IPC lifecycle:
//   - If the shell is not running when an IPC call is made, it is auto-started
//     (except for close/hide/deactivate/dismiss functions, which are no-ops).
//   - If the first IPC call fails while instances exist, we retry once after 500ms
//     to handle deferred panel loading at boot.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var ipcCommandRunner = runQSIPCCommand

// Overridable for testing — skip actual qs binary calls.
var ipcInstanceChecker = hasInstances
var ipcStarter = startBackground

// Functions that should NOT trigger an auto-start when the shell isn't running.
var noAutoStartFunctions = map[string]bool{
	"close":      true,
	"hide":       true,
	"deactivate": true,
	"dismiss":    true,
}

func runIPCCommand(configDir string, callArgs []string, functionName string) error {
	runtimeDir, err := resolveIPCRuntimeDir(configDir)
	if err != nil {
		return err
	}

	// Lifecycle: ensure shell is running before calling IPC.
	// Skip for functions that are no-ops when the shell is down.
	if functionName == "" || !noAutoStartFunctions[functionName] {
		if err := ensureRunningInstanceForIPC(runtimeDir, functionName); err != nil {
			return fmt.Errorf("shell not available: %w", err)
		}
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
	binary := resolveQSBinary()

	// First attempt
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err == nil {
		return nil
	}

	// If instances exist, retry once after a brief wait.
	// IPC handlers register ~500ms after shell start (deferred panel loading).
	if hasInstances(configDir) {
		time.Sleep(500 * time.Millisecond)
		cmd2 := exec.Command(binary, args...)
		cmd2.Stdin = os.Stdin
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr
		return cmd2.Run()
	}

	return fmt.Errorf("ipc call failed")
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

// ---------------------------------------------------------------------------
// Instance detection and auto-start
// ---------------------------------------------------------------------------

func hasInstances(configDir string) bool {
	binary := resolveQSBinary()
	out, err := exec.Command(binary, "-p", configDir, "list").CombinedOutput()
	if err != nil {
		return false
	}
	text := strings.TrimSpace(string(out))
	// qs list returns "No running instances" when none exist
	return text != "" && !strings.Contains(text, "No running instances")
}

func ensureRunningInstanceForIPC(configDir string, functionName string) error {
	if ipcInstanceChecker(configDir) {
		return nil
	}

	// If the shell isn't running and the function is a "close" variant,
	// there's nothing to do — the panel is already closed.
	if functionName != "" && noAutoStartFunctions[functionName] {
		return nil
	}

	// Start the shell in the background
	return ipcStarter(configDir)
}

func startBackground(configDir string) error {
	binary := resolveQSBinary()
	cmd := exec.Command(binary, "-n", "-d", "-p", configDir)
	// Detach from terminal
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Wait briefly for the shell to register its IPC handlers
	return waitForStart(configDir, 60, 100*time.Millisecond)
}

func waitForStart(configDir string, maxPolls int, pollDelay time.Duration) error {
	for i := 0; i < maxPolls; i++ {
		if hasInstances(configDir) {
			return nil
		}
		time.Sleep(pollDelay)
	}
	return fmt.Errorf("timed out waiting for shell to start")
}