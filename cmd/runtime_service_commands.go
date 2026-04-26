// File: runtime_service_commands.go
//
// Runtime and service lifecycle commands for upstream parity.
//
// Commands added:
//   - run: start Quickshell with the resolved iNiR runtime payload
//   - service: install/enable/start/stop/restart/status/logs user service wiring
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	serviceExecutableResolver = resolveServiceLauncherPath
	serviceSystemctlRunner    = runSystemctlUser
	serviceJournalctlRunner   = runJournalctlUser
)

func init() {
	rootCmd.AddCommand(newRunCommand())
	rootCmd.AddCommand(newServiceCommand())
	rootCmd.AddCommand(newCleanupOrphansCommand())
}

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "run [-c PATH] [--session]",
		Short:              "Run Quickshell with iNiR runtime payload",
		DisableFlagParsing: true,
		RunE:               runRunCommand,
	}
}

func runRunCommand(cmd *cobra.Command, args []string) error {
	configPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("missing path after %s", args[i])
			}
			configPath = args[i+1]
			i++
		case "--session":
			// compatibility flag accepted for upstream parity
		case "-h", "--help":
			fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli run [-c PATH] [--session]")
			return nil
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	runtimeDir, err := resolveIPCRuntimeDir(configPath)
	if err != nil {
		return err
	}

	qsBin := resolveQSBinary()
	execCmd := exec.Command(qsBin, "-p", runtimeDir)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func newServiceCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "service <install|uninstall|enable|disable|start|stop|restart|status|logs> [args...]",
		Short:              "Manage inir user systemd service",
		DisableFlagParsing: true,
		RunE:               runServiceCommand,
	}
}

func newCleanupOrphansCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup-orphans",
		Short: "Cleanup orphaned shell artifacts (compatibility command)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Compatibility entrypoint used by systemd ExecStopPost.
			// Upstream performs deep orphan cleanup in bash; here we keep this
			// command available so service stop hooks do not fail when aliasing
			// `inir` to inir-cli.
			return nil
		},
	}
}

func runServiceCommand(cmd *cobra.Command, args []string) error {
	action := "status"
	rest := []string{}
	if len(args) > 0 {
		action = args[0]
		rest = args[1:]
	}

	switch action {
	case "-h", "--help":
		fmt.Fprintln(cmd.OutOrStdout(), "Usage: inir-cli service <install|uninstall|enable|disable|start|stop|restart|status|logs> [args...]")
		return nil
	case "install":
		path, err := installUserService()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	case "uninstall", "remove":
		_ = disableInirService()
		servicePath := filepath.Join(systemdUserDir(), "inir.service")
		if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		_ = serviceSystemctlRunner([]string{"daemon-reload"})
		fmt.Fprintln(cmd.OutOrStdout(), "Uninstalled inir.service")
		return nil
	case "enable":
		return enableInirService()
	case "disable":
		return disableInirService()
	case "start":
		if _, err := installUserService(); err != nil {
			return err
		}
		return serviceSystemctlRunner(append([]string{"start", "inir.service"}, rest...))
	case "stop":
		return serviceSystemctlRunner(append([]string{"stop", "inir.service"}, rest...))
	case "restart":
		if _, err := installUserService(); err != nil {
			return err
		}
		return serviceSystemctlRunner(append([]string{"restart", "inir.service"}, rest...))
	case "status":
		return serviceSystemctlRunner(append([]string{"status", "inir.service"}, rest...))
	case "logs":
		return serviceJournalctlRunner(append([]string{"--user-unit", "inir.service"}, rest...))
	default:
		return fmt.Errorf("unknown service action: %s", action)
	}
}

func installUserService() (string, error) {
	launcher := serviceExecutableResolver()
	if launcher == "" {
		return "", fmt.Errorf("unable to resolve launcher path")
	}

	userDir := systemdUserDir()
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", err
	}

	servicePath := filepath.Join(userDir, "inir.service")
	if err := os.WriteFile(servicePath, []byte(renderServiceUnit(launcher)), 0644); err != nil {
		return "", err
	}

	if err := serviceSystemctlRunner([]string{"daemon-reload"}); err != nil {
		return "", err
	}

	return servicePath, nil
}

func enableInirService() error {
	if _, err := installUserService(); err != nil {
		return err
	}

	target, err := detectCompositorService()
	if err != nil {
		return err
	}

	wantsDir := filepath.Join(systemdUserDir(), target+".wants")
	if err := os.MkdirAll(wantsDir, 0755); err != nil {
		return err
	}

	servicePath := filepath.Join(systemdUserDir(), "inir.service")
	linkPath := filepath.Join(wantsDir, "inir.service")
	if err := os.Symlink(servicePath, linkPath); err != nil {
		if !os.IsExist(err) {
			if err := os.Remove(linkPath); err != nil {
				return err
			}
			if err := os.Symlink(servicePath, linkPath); err != nil {
				return err
			}
		}
	}

	if err := serviceSystemctlRunner([]string{"daemon-reload"}); err != nil {
		return err
	}

	fmt.Printf("Enabled inir.service (wired to %s)\n", target)
	return nil
}

func disableInirService() error {
	_ = serviceSystemctlRunner([]string{"stop", "inir.service"})

	entries, err := os.ReadDir(systemdUserDir())
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("inir.service was not enabled")
			return nil
		}
		return err
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".wants") {
			continue
		}
		linkPath := filepath.Join(systemdUserDir(), entry.Name(), "inir.service")
		if _, err := os.Lstat(linkPath); err == nil {
			_ = os.Remove(linkPath)
			found = true
		}
	}

	_ = serviceSystemctlRunner([]string{"daemon-reload"})
	if found {
		fmt.Println("Disabled inir.service")
	} else {
		fmt.Println("inir.service was not enabled")
	}
	return nil
}

func detectCompositorService() (string, error) {
	if serviceUnitExists("niri.service") {
		return "niri.service", nil
	}
	if serviceUnitExists("wayland-wm@Hyprland.service") {
		return "wayland-wm@Hyprland.service", nil
	}
	return "", fmt.Errorf("no supported compositor detected (niri or Hyprland)")
}

func serviceUnitExists(unit string) bool {
	return serviceSystemctlRunner([]string{"cat", unit}) == nil
}

func renderServiceUnit(launcher string) string {
	escaped := strings.ReplaceAll(launcher, `\`, `\\`)
	return fmt.Sprintf(`[Unit]
Description=iNiR shell
PartOf=graphical-session.target
After=graphical-session.target
Requisite=graphical-session.target
StartLimitIntervalSec=30
StartLimitBurst=3

[Service]
Type=simple
Environment=QT_SCALE_FACTOR=1
Environment=QT_LOGGING_RULES=quickshell.dbus.properties=false;qt.qml.settings.warning=false;qt.core.qsettings.warning=false;kf.xmlgui=false;kf.coreaddons=false;kf.config.core=false;kf.iconthemes=false
ExecStart=%s run --session
SuccessExitStatus=143
KillMode=process
KillSignal=SIGTERM
Restart=on-failure
RestartSec=5
TimeoutStopSec=15
LimitCORE=0
ExecStopPost=-%s cleanup-orphans
IOSchedulingPriority=2
`, escaped, escaped)
}

func systemdUserDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "systemd", "user")
}

func resolveServiceLauncherPath() string {
	if override := os.Getenv("INIR_LAUNCHER_PATH"); override != "" {
		return override
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return exe
}

func runSystemctlUser(args []string) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl not found")
	}
	full := append([]string{"--user"}, args...)
	execCmd := exec.Command("systemctl", full...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func runJournalctlUser(args []string) error {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return fmt.Errorf("journalctl not found")
	}
	execCmd := exec.Command("journalctl", args...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}
