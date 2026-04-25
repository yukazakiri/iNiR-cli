package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseScaffoldEnv(t *testing.T) {
	t.Parallel()

	parsed, err := parseScaffoldEnv([]string{"A=1", "B=value"})
	if err != nil {
		t.Fatalf("parseScaffoldEnv error: %v", err)
	}
	if parsed["A"] != "1" || parsed["B"] != "value" {
		t.Fatalf("unexpected parsed env map: %#v", parsed)
	}

	if _, err := parseScaffoldEnv([]string{"INVALID"}); err == nil {
		t.Fatalf("expected parse error for invalid entry")
	}
}

func TestThemeScaffoldTargetCreatesJSONSpec(t *testing.T) {
	targetsDir := t.TempDir()

	originalConfig := flagConfig
	originalDir := flagScaffoldDir
	originalCommand := flagScaffoldCommand
	originalDesc := flagScaffoldDescription
	originalArgs := flagScaffoldArgs
	originalInputs := flagScaffoldInputs
	originalEnv := flagScaffoldEnv
	originalEnabled := flagScaffoldEnabled
	originalForce := flagScaffoldForce
	t.Cleanup(func() {
		flagConfig = originalConfig
		flagScaffoldDir = originalDir
		flagScaffoldCommand = originalCommand
		flagScaffoldDescription = originalDesc
		flagScaffoldArgs = originalArgs
		flagScaffoldInputs = originalInputs
		flagScaffoldEnv = originalEnv
		flagScaffoldEnabled = originalEnabled
		flagScaffoldForce = originalForce
	})

	flagConfig = filepath.Join(t.TempDir(), "config.json")
	flagScaffoldDir = targetsDir
	flagScaffoldCommand = "/usr/bin/demo-apply"
	flagScaffoldDescription = "Demo app target"
	flagScaffoldArgs = []string{"--mode", "material"}
	flagScaffoldInputs = []string{"palette.json", "terminal.json"}
	flagScaffoldEnv = []string{"DEMO=1"}
	flagScaffoldEnabled = true
	flagScaffoldForce = false

	if err := runThemeScaffoldTarget(themeScaffoldTargetCmd, []string{"demo-app"}); err != nil {
		t.Fatalf("runThemeScaffoldTarget error: %v", err)
	}

	specPath := filepath.Join(targetsDir, "demo-app.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read scaffolded spec: %v", err)
	}

	var spec externalThemeTarget
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("unmarshal scaffolded spec: %v", err)
	}

	if spec.ID != "demo-app" || spec.Command != "/usr/bin/demo-apply" {
		t.Fatalf("unexpected scaffolded spec: %#v", spec)
	}
	if len(spec.Args) != 2 || spec.Args[0] != "--mode" {
		t.Fatalf("unexpected args in scaffolded spec: %#v", spec.Args)
	}
	if spec.Env["DEMO"] != "1" {
		t.Fatalf("unexpected env in scaffolded spec: %#v", spec.Env)
	}
}
