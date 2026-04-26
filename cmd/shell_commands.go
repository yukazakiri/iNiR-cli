// File: shell_commands.go
//
// Non-IPC shell commands that execute direct system actions.
// These mirror upstream inir script behavior for commands that are not
// Quickshell IPC targets.
//
// Commands:
//   - terminal:     Launch the configured terminal emulator
//   - close-window: Close the focused window (with QS confirm dialog support)
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTerminalCommand())
	rootCmd.AddCommand(newCloseWindowCommand())
	rootCmd.AddCommand(newBrowserCommand())
}

// ---------------------------------------------------------------------------
// terminal
// ---------------------------------------------------------------------------

func newTerminalCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "terminal [-c PATH] [args...]",
		Short:              "Launch the configured terminal emulator",
		DisableFlagParsing: true,
		RunE:               runTerminalCommand,
	}
}

func runTerminalCommand(cmd *cobra.Command, args []string) error {
	configPath := ""
	rest := args

	// Parse -c/--config prefix
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("missing path after %s", args[i])
			}
			configPath = args[i+1]
			i++
		case "--":
			rest = args[i+1:]
			break
		case "-h", "--help":
			fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli terminal [-c PATH] [args...]")
			return nil
		default:
			rest = args[i:]
			break
		}
	}

	terminal := resolveTerminalFromConfig(configPath)
	if terminal == "" {
		terminal = findTerminalFallback()
	}
	if terminal == "" {
		return fmt.Errorf("no terminal emulator found")
	}

	execCmd := exec.Command(terminal, rest...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func resolveTerminalFromConfig(configPath string) string {
	if configPath == "" {
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(os.Getenv("HOME"), ".config")
		}
		configPath = filepath.Join(configHome, "inir", "config.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}

	// Check root.terminal first
	if t, ok := root["terminal"].(string); ok && t != "" {
		return t
	}

	// Check appearance.terminal
	if appearance, ok := root["appearance"].(map[string]interface{}); ok {
		if t, ok := appearance["terminal"].(string); ok && t != "" {
			return t
		}
	}

	return ""
}

func findTerminalFallback() string {
	// Configured terminal not found or not available — try fallback chain
	for _, candidate := range []string{"kitty", "foot", "ghostty", "alacritty", "wezterm", "konsole", "xterm"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// close-window
// ---------------------------------------------------------------------------

func newCloseWindowCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "close-window [-c PATH]",
		Short:              "Close the focused window",
		DisableFlagParsing: true,
		RunE:               runCloseWindowCommand,
	}
}

func runCloseWindowCommand(cmd *cobra.Command, args []string) error {
	// Parse -c/--config prefix (ignored, but accepted for compatibility)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("missing path after %s", args[i])
			}
			i++
		case "-h", "--help":
			fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli close-window [-c PATH]")
			return nil
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	// Capture focused window info immediately (race condition protection)
	focusedID, focusedAppID, err := captureFocusedWindow()
	if err != nil {
		// Niri not available — can't close window
		return fmt.Errorf("cannot determine focused window: %w", err)
	}

	// If QS/quickshell is not running, close directly
	if !isQShellRunning() {
		return closeFocusedWindow(focusedID, focusedAppID)
	}

	// QS is running — try IPC closeConfirm trigger first
	if tryCloseConfirmTrigger() {
		return nil
	}

	// Fallback — IPC failed or timed out. Close the originally captured window.
	return closeFocusedWindow(focusedID, focusedAppID)
}

func captureFocusedWindow() (id int, appID string, err error) {
	out, err := exec.Command("niri", "msg", "-j", "focused-window").Output()
	if err != nil {
		return 0, "", err
	}

	var win map[string]interface{}
	if err := json.Unmarshal(out, &win); err != nil {
		return 0, "", err
	}

	if idRaw, ok := win["id"].(float64); ok {
		id = int(idRaw)
	}
	if app, ok := win["app_id"].(string); ok {
		appID = app
	}

	return id, appID, nil
}

func isQShellRunning() bool {
	if _, err := exec.LookPath("pgrep"); err != nil {
		// Can't check — assume not running
		return false
	}
	cmd := exec.Command("pgrep", "-x", "qs")
	if err := cmd.Run(); err == nil {
		return true
	}
	cmd = exec.Command("pgrep", "-x", "quickshell")
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

func tryCloseConfirmTrigger() bool {
	// Try qs ipc call closeConfirm trigger with a 1s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Try to find runtime dir for qs
	runtimeDir, _ := resolveIPCRuntimeDir("")
	if runtimeDir == "" {
		return false
	}

	qsBin := resolveQSBinary()
	execCmd := exec.CommandContext(ctx, qsBin, "-p", runtimeDir, "ipc", "call", "closeConfirm", "trigger")
	err := execCmd.Run()
	return err == nil
}

func closeFocusedWindow(focusedID int, appID string) error {
	// Special case: Spotify — move to workspace 99 instead of closing
	if strings.ToLower(appID) == "spotify" && focusedID != 0 {
		cmd := exec.Command("niri", "msg", "action", "move-window-to-workspace",
			"--window-id", fmt.Sprintf("%d", focusedID),
			"--focus", "false", "99")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if focusedID != 0 {
		cmd := exec.Command("niri", "msg", "action", "close-window", "--id", fmt.Sprintf("%d", focusedID))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command("niri", "msg", "action", "close-window")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ---------------------------------------------------------------------------
// browser
// ---------------------------------------------------------------------------

func newBrowserCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "browser [-c PATH] [url]",
		Short:              "Launch the configured web browser",
		DisableFlagParsing: true,
		RunE:               runBrowserCommand,
	}
}

func runBrowserCommand(cmd *cobra.Command, args []string) error {
	configPath := ""
	rest := args

	// Parse -c/--config prefix
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("missing path after %s", args[i])
			}
			configPath = args[i+1]
			i++
		case "--":
			rest = args[i+1:]
			break
		case "-h", "--help":
			fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli browser [-c PATH] [url]")
			return nil
		default:
			rest = args[i:]
			break
		}
	}

	browser := resolveBrowserFromConfig(configPath)
	if browser == "" {
		browser = "xdg-open"
	}

	execCmd := exec.Command(browser, rest...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func resolveBrowserFromConfig(configPath string) string {
	if configPath == "" {
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(os.Getenv("HOME"), ".config")
		}
		configPath = filepath.Join(configHome, "inir", "config.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}

	// Check .apps.browser
	if apps, ok := root["apps"].(map[string]interface{}); ok {
		if b, ok := apps["browser"].(string); ok && b != "" {
			return b
		}
	}

	return ""
}