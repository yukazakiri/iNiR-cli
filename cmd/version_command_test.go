package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunVersionCommandJSONFromLocalVersionFile(t *testing.T) {
	origSetupResolver := setupDirResolver
	defer func() {
		setupDirResolver = origSetupResolver
	}()

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("INIR_RUNTIME_DIR", "")

	versionDir := filepath.Join(tmp, ".config", "inir")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("mkdir version dir: %v", err)
	}
	versionPayload := map[string]string{
		"version":        "1.2.3",
		"commit":         "abc123",
		"installMode":    "repo-copy",
		"updateStrategy": "git",
		"source":         "setup-update",
	}
	encoded, _ := json.Marshal(versionPayload)
	if err := os.WriteFile(filepath.Join(versionDir, "version.json"), encoded, 0644); err != nil {
		t.Fatalf("write version.json: %v", err)
	}

	setupDirResolver = func() (string, error) {
		return "", nil
	}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	if err := runVersionCommand(cmd, []string{"--json"}); err != nil {
		t.Fatalf("runVersionCommand returned error: %v", err)
	}

	text := strings.TrimSpace(out.String())
	if !strings.Contains(text, "\"version\":\"1.2.3\"") {
		t.Fatalf("expected raw local version.json output, got: %s", text)
	}
}

func TestRunVersionCommandUnknownFlag(t *testing.T) {
	cmd := &cobra.Command{}
	err := runVersionCommand(cmd, []string{"--wat"})
	if err == nil {
		t.Fatalf("expected unknown option error")
	}
}
