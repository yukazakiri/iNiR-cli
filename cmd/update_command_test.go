package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunUpdateCommandStripsConfigFlag(t *testing.T) {
	origResolver := setupDirResolver
	origRunner := setupCommandRunner
	origCLIUpdater := cliUpdateRunner
	defer func() {
		setupDirResolver = origResolver
		setupCommandRunner = origRunner
		cliUpdateRunner = origCLIUpdater
	}()

	setupDirResolver = func() (string, error) {
		return "/tmp/inir", nil
	}

	var gotDir string
	var gotArgs []string
	setupCommandRunner = func(dir string, args []string) error {
		gotDir = dir
		gotArgs = append([]string{}, args...)
		return nil
	}
	cliUpdateRunner = func(*cobra.Command, string) error {
		t.Fatalf("cli update should not be called for upstream flow")
		return nil
	}

	err := runUpdateCommand(&cobra.Command{}, []string{"-c", "/tmp/config", "--local", "-y", "-q"})
	if err != nil {
		t.Fatalf("runUpdateCommand returned error: %v", err)
	}

	if gotDir != "/tmp/inir" {
		t.Fatalf("expected setup dir /tmp/inir, got %q", gotDir)
	}

	wantArgs := []string{"update", "--local", "-y", "-q"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("forwarded args mismatch\nwant: %#v\n got: %#v", wantArgs, gotArgs)
	}
}

func TestRunUpdateCommandMissingConfigPath(t *testing.T) {
	err := runUpdateCommand(&cobra.Command{}, []string{"-c"})
	if err == nil {
		t.Fatalf("expected error for missing -c value")
	}
}

func TestRunUpdateCommandCLIFlow(t *testing.T) {
	origResolver := setupDirResolver
	origRunner := setupCommandRunner
	origCLIUpdater := cliUpdateRunner
	defer func() {
		setupDirResolver = origResolver
		setupCommandRunner = origRunner
		cliUpdateRunner = origCLIUpdater
	}()

	setupDirResolver = func() (string, error) {
		t.Fatalf("setup resolver should not run for --cli")
		return "", nil
	}
	setupCommandRunner = func(string, []string) error {
		t.Fatalf("upstream setup runner should not run for --cli")
		return nil
	}

	called := false
	gotVersion := ""
	cliUpdateRunner = func(_ *cobra.Command, version string) error {
		called = true
		gotVersion = version
		return nil
	}

	err := runUpdateCommand(&cobra.Command{}, []string{"--cli", "--version", "v1.2.3"})
	if err != nil {
		t.Fatalf("runUpdateCommand returned error: %v", err)
	}

	if !called {
		t.Fatalf("expected cli updater to be called")
	}
	if gotVersion != "v1.2.3" {
		t.Fatalf("expected version v1.2.3, got %q", gotVersion)
	}
}

func TestRunUpdateCommandVersionWithoutCLI(t *testing.T) {
	err := runUpdateCommand(&cobra.Command{}, []string{"--version", "v1.2.3"})
	if err == nil {
		t.Fatalf("expected error when --version is used without --cli")
	}
}

func TestResolveSetupDirFromVersionFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("HOME", tmp)

	repoDir := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "setup"), []byte("#!/usr/bin/env bash\n"), 0755); err != nil {
		t.Fatalf("write setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "shell.qml"), []byte("// stub\n"), 0644); err != nil {
		t.Fatalf("write shell.qml: %v", err)
	}

	versionDir := filepath.Join(tmp, ".config", "inir")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("mkdir version dir: %v", err)
	}
	versionJSON := map[string]string{"repoPath": repoDir}
	encoded, _ := json.Marshal(versionJSON)
	if err := os.WriteFile(filepath.Join(versionDir, "version.json"), encoded, 0644); err != nil {
		t.Fatalf("write version.json: %v", err)
	}

	resolved, err := resolveSetupDir()
	if err != nil {
		t.Fatalf("resolveSetupDir returned error: %v", err)
	}
	if resolved != repoDir {
		t.Fatalf("expected %q, got %q", repoDir, resolved)
	}
}
