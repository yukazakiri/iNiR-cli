package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeIPCTargetSupportsKebabAliases(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"control-panel":      "controlPanel",
		"wallpaper-selector": "wallpaperSelector",
		"sidebar-right":      "sidebarRight",
		"overview":           "overview",
	}

	for input, expected := range tests {
		actual, ok := normalizeIPCTarget(input)
		if !ok {
			t.Fatalf("expected %q to be recognized", input)
		}
		if actual != expected {
			t.Fatalf("normalizeIPCTarget(%q) = %q, expected %q", input, actual, expected)
		}
	}
}

func TestValidateIPCFunctionRejectsUnknownFunction(t *testing.T) {
	t.Parallel()

	target, ok := findIPCTarget("overview")
	if !ok {
		t.Fatalf("overview target missing")
	}
	if err := validateIPCFunction(target, "toggle"); err != nil {
		t.Fatalf("expected overview toggle to validate: %v", err)
	}

	err := validateIPCFunction(target, "nope")
	if err == nil {
		t.Fatalf("expected unknown function to fail")
	}
	if !strings.Contains(err.Error(), "overview") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected contextual error, got %v", err)
	}
}

func TestParseIPCPrefixArgsStopsAtFunction(t *testing.T) {
	t.Parallel()

	parsed, err := parseIPCPrefixArgs([]string{"-c", "/tmp/inir", "overview", "toggle", "--literal"})
	if err != nil {
		t.Fatalf("parseIPCPrefixArgs returned error: %v", err)
	}
	if parsed.Config != "/tmp/inir" {
		t.Fatalf("expected config path, got %q", parsed.Config)
	}
	expectedRest := []string{"overview", "toggle", "--literal"}
	if !reflect.DeepEqual(parsed.Rest, expectedRest) {
		t.Fatalf("rest = %#v, expected %#v", parsed.Rest, expectedRest)
	}
}

func TestRunIPCCommandUsesRuntimeDirAndCallArgs(t *testing.T) {
	runtimeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runtimeDir, "shell.qml"), []byte("import QtQuick"), 0644); err != nil {
		t.Fatalf("write shell.qml fixture: %v", err)
	}

	originalRunner := ipcCommandRunner
	t.Cleanup(func() { ipcCommandRunner = originalRunner })

	var capturedDir string
	var capturedArgs []string
	ipcCommandRunner = func(configDir string, callArgs []string) error {
		capturedDir = configDir
		capturedArgs = append([]string{}, callArgs...)
		return nil
	}

	err := runIPCCommand(runtimeDir, []string{"overview", "toggle"})
	if err != nil {
		t.Fatalf("runIPCCommand returned error: %v", err)
	}
	if capturedDir != runtimeDir {
		t.Fatalf("capturedDir = %q, expected %q", capturedDir, runtimeDir)
	}
	expectedArgs := []string{"overview", "toggle"}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Fatalf("capturedArgs = %#v, expected %#v", capturedArgs, expectedArgs)
	}
}

func TestRawIPCCommandDoesNotNormalizeOrValidateTarget(t *testing.T) {
	runtimeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runtimeDir, "shell.qml"), []byte("import QtQuick"), 0644); err != nil {
		t.Fatalf("write shell.qml fixture: %v", err)
	}

	originalRunner := ipcCommandRunner
	t.Cleanup(func() { ipcCommandRunner = originalRunner })

	var capturedArgs []string
	ipcCommandRunner = func(configDir string, callArgs []string) error {
		capturedArgs = append([]string{}, callArgs...)
		return nil
	}

	command := newIPCRootCommand()
	command.SetArgs([]string{"-c", runtimeDir, "control-panel", "futureFunction"})
	if err := command.Execute(); err != nil {
		t.Fatalf("raw ipc command returned error: %v", err)
	}

	expectedArgs := []string{"control-panel", "futureFunction"}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Fatalf("capturedArgs = %#v, expected %#v", capturedArgs, expectedArgs)
	}
}

func TestSettingsCommandDefaultsToOpen(t *testing.T) {
	runtimeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runtimeDir, "shell.qml"), []byte("import QtQuick"), 0644); err != nil {
		t.Fatalf("write shell.qml fixture: %v", err)
	}

	originalRunner := ipcCommandRunner
	t.Cleanup(func() { ipcCommandRunner = originalRunner })

	var capturedArgs []string
	ipcCommandRunner = func(configDir string, callArgs []string) error {
		capturedArgs = append([]string{}, callArgs...)
		return nil
	}

	target, ok := findIPCTarget("settings")
	if !ok {
		t.Fatalf("settings target missing")
	}
	command := newIPCTargetCommand(target)
	command.SetArgs([]string{"-c", runtimeDir})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	if err := command.Execute(); err != nil {
		t.Fatalf("settings command returned error: %v", err)
	}

	expectedArgs := []string{"settings", "open"}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Fatalf("capturedArgs = %#v, expected %#v", capturedArgs, expectedArgs)
	}
}

func TestTargetCommandRunsCanonicalTargetFunction(t *testing.T) {
	runtimeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runtimeDir, "shell.qml"), []byte("import QtQuick"), 0644); err != nil {
		t.Fatalf("write shell.qml fixture: %v", err)
	}

	originalRunner := ipcCommandRunner
	t.Cleanup(func() { ipcCommandRunner = originalRunner })

	var capturedArgs []string
	ipcCommandRunner = func(configDir string, callArgs []string) error {
		capturedArgs = append([]string{}, callArgs...)
		return nil
	}

	target, ok := findIPCTarget("controlPanel")
	if !ok {
		t.Fatalf("controlPanel target missing")
	}
	command := newIPCTargetCommand(target)
	command.SetArgs([]string{"-c", runtimeDir, "toggle"})
	if err := command.Execute(); err != nil {
		t.Fatalf("target command returned error: %v", err)
	}

	expectedArgs := []string{"controlPanel", "toggle"}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Fatalf("capturedArgs = %#v, expected %#v", capturedArgs, expectedArgs)
	}
}

func TestResolveIPCRuntimeDirRejectsInvalidPath(t *testing.T) {
	t.Parallel()

	invalidDir := t.TempDir()
	_, err := resolveIPCRuntimeDir(invalidDir)
	if err == nil {
		t.Fatalf("expected invalid runtime dir error")
	}
	if !strings.Contains(err.Error(), "shell.qml") {
		t.Fatalf("expected shell.qml error, got %v", err)
	}
}

func TestPrintIPCTargetHelpIncludesFunctionAndFamily(t *testing.T) {
	t.Parallel()

	target, ok := findIPCTarget("overview")
	if !ok {
		t.Fatalf("overview target missing")
	}

	var buffer bytes.Buffer
	printIPCTargetHelp(&buffer, target)
	output := buffer.String()

	if !strings.Contains(output, "overview") || !strings.Contains(output, "Available functions") {
		t.Fatalf("unexpected help output: %s", output)
	}
	if !strings.Contains(output, "toggle") || !strings.Contains(output, "Family: shared") {
		t.Fatalf("expected function/family in help output: %s", output)
	}
}
