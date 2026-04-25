package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukazakiri/inir-cli/internal/config"
)

func TestApplyThemeTargetsRunsExternalCommandTarget(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("INIR_THEME_TARGETS_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(configDir, "xdg-config"))
	configPath := filepath.Join(configDir, "config.json")
	targetsDir := filepath.Join(configDir, "targets")
	outputDir := filepath.Join(configDir, "generated")

	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(targetsDir, 0755); err != nil {
		t.Fatalf("create targets dir: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}

	spec := `{"id":"demo","type":"command","command":"/bin/demo","args":["--flag"]}`
	if err := os.WriteFile(filepath.Join(targetsDir, "demo.json"), []byte(spec), 0644); err != nil {
		t.Fatalf("write target spec: %v", err)
	}

	contract := newOutputContract(outputDir)
	originalRunner := externalTargetCommandRunner
	t.Cleanup(func() { externalTargetCommandRunner = originalRunner })

	var called bool
	var capturedCommand string
	var capturedArgs []string
	var capturedEnv []string
	externalTargetCommandRunner = func(command string, args []string, env []string) error {
		called = true
		capturedCommand = command
		capturedArgs = append([]string{}, args...)
		capturedEnv = append([]string{}, env...)
		return nil
	}

	err := applyThemeTargets(config.DefaultConfig(), configPath, contract, []string{"demo"})
	if err != nil {
		t.Fatalf("applyThemeTargets returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected external target command runner invocation")
	}
	if capturedCommand != "/bin/demo" {
		t.Fatalf("unexpected command %q", capturedCommand)
	}
	if len(capturedArgs) != 1 || capturedArgs[0] != "--flag" {
		t.Fatalf("unexpected args %#v", capturedArgs)
	}

	joinedEnv := strings.Join(capturedEnv, "\n")
	if !strings.Contains(joinedEnv, "INIR_OUTPUT_DIR="+outputDir) {
		t.Fatalf("missing INIR_OUTPUT_DIR env: %s", joinedEnv)
	}
	if !strings.Contains(joinedEnv, "INIR_CONFIG_JSON="+configPath) {
		t.Fatalf("missing INIR_CONFIG_JSON env: %s", joinedEnv)
	}
}

func TestApplyThemeTargetsReturnsErrorForUnknownTarget(t *testing.T) {
	t.Setenv("INIR_THEME_TARGETS_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-config"))
	contract := newOutputContract(t.TempDir())
	err := applyThemeTargets(config.DefaultConfig(), filepath.Join(t.TempDir(), "config.json"), contract, []string{"missing-target"})
	if err == nil {
		t.Fatalf("expected unknown target error")
	}
	if !strings.Contains(err.Error(), "target not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
