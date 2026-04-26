package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderServiceUnitIncludesLauncherPath(t *testing.T) {
	t.Parallel()

	unit := renderServiceUnit("/usr/bin/inir-cli")
	if !strings.Contains(unit, "ExecStart=/usr/bin/inir-cli run --session") {
		t.Fatalf("ExecStart not rendered correctly: %s", unit)
	}
	if !strings.Contains(unit, "ExecStopPost=-/usr/bin/inir-cli cleanup-orphans") {
		t.Fatalf("ExecStopPost not rendered correctly: %s", unit)
	}
}

func TestEnableInirServiceCreatesCompositorWantsLink(t *testing.T) {
	origSystemctl := serviceSystemctlRunner
	origLauncher := serviceExecutableResolver
	defer func() {
		serviceSystemctlRunner = origSystemctl
		serviceExecutableResolver = origLauncher
	}()

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	serviceExecutableResolver = func() string {
		return "/usr/bin/inir-cli"
	}

	serviceSystemctlRunner = func(args []string) error {
		if len(args) >= 2 && args[0] == "cat" && args[1] == "niri.service" {
			return nil
		}
		if len(args) >= 2 && args[0] == "cat" && args[1] == "wayland-wm@Hyprland.service" {
			return os.ErrNotExist
		}
		return nil
	}

	if err := enableInirService(); err != nil {
		t.Fatalf("enableInirService returned error: %v", err)
	}

	linkPath := filepath.Join(tmp, ".config", "systemd", "user", "niri.service.wants", "inir.service")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("expected wants symlink to exist: %v", err)
	}

	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatalf("resolve wants symlink: %v", err)
	}
	want := filepath.Join(tmp, ".config", "systemd", "user", "inir.service")
	if resolved != want {
		t.Fatalf("wants link target mismatch: want %q got %q", want, resolved)
	}
}
