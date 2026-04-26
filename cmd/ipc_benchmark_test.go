package cmd

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const defaultInirScriptPath = "/home/admin/inir/scripts/inir"

// BenchmarkIPCTargetHelpGo measures in-process Go command dispatch/help path.
func BenchmarkIPCTargetHelpGo(b *testing.B) {
	targetMeta, ok := findIPCTarget("overview")
	if !ok {
		b.Fatal("overview target metadata not found")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := newIPCTargetCommand(targetMeta)
		command.SetArgs([]string{"--help"})
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		if err := command.Execute(); err != nil {
			b.Fatalf("go target help failed: %v", err)
		}
	}
}

// BenchmarkIPCTargetHelpDefaultInir measures external Bash launcher dispatch/help path.
// This compares command-layer overhead only (help path), not actual IPC call runtime.
func BenchmarkIPCTargetHelpDefaultInir(b *testing.B) {
	if _, err := os.Stat(defaultInirScriptPath); err != nil {
		b.Skipf("default inir script not available: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := exec.Command("bash", defaultInirScriptPath, "overview", "--help")
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if err := command.Run(); err != nil {
			b.Fatalf("default inir target help failed: %v", err)
		}
	}
}

// BenchmarkIPCRealCallGo measures a real IPC call path through inir-cli
// using a fake qs binary and runtime payload.
func BenchmarkIPCRealCallGo(b *testing.B) {
	testEnv := mustPrepareIPCBenchmarkEnv(b)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := exec.Command(testEnv.cliPath, "overview", "-c", testEnv.runtimeDir, "toggle")
		command.Env = testEnv.env
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if err := command.Run(); err != nil {
			b.Fatalf("go real ipc call failed: %v", err)
		}
	}
}

// BenchmarkIPCRealCallDefaultInir measures the upstream inir launcher IPC call path
// with the same fake qs binary and runtime payload.
func BenchmarkIPCRealCallDefaultInir(b *testing.B) {
	if _, err := os.Stat(defaultInirScriptPath); err != nil {
		b.Skipf("default inir script not available: %v", err)
	}
	testEnv := mustPrepareIPCBenchmarkEnv(b)
	if _, err := os.Stat("/usr/bin/qs"); err != nil {
		b.Skipf("/usr/bin/qs unavailable for upstream inir benchmark: %v", err)
	}

	preflight := exec.Command("bash", defaultInirScriptPath, "overview", "-c", testEnv.runtimeDir, "toggle")
	preflight.Env = testEnv.env
	preflight.Stdout = io.Discard
	preflight.Stderr = io.Discard
	if err := preflight.Run(); err != nil {
		b.Skipf("upstream inir preflight failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := exec.Command("bash", defaultInirScriptPath, "overview", "-c", testEnv.runtimeDir, "toggle")
		command.Env = testEnv.env
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if err := command.Run(); err != nil {
			b.Fatalf("default inir real ipc call failed: %v", err)
		}
	}
}

type ipcBenchmarkEnv struct {
	cliPath    string
	runtimeDir string
	env        []string
}

func mustPrepareIPCBenchmarkEnv(b *testing.B) ipcBenchmarkEnv {
	b.Helper()

	root := b.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		b.Fatalf("create runtime dir: %v", err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		b.Fatalf("create bin dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(runtimeDir, "shell.qml"), []byte("import QtQuick"), 0644); err != nil {
		b.Fatalf("write shell.qml: %v", err)
	}

	qsScript := filepath.Join(binDir, "qs")
	qsBody := "#!/usr/bin/env bash\n" +
		"if [[ \"$3\" == \"list\" ]]; then\n" +
		"  echo \"instance-1\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [[ \"$3\" == \"ipc\" ]]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(qsScript, []byte(qsBody), 0755); err != nil {
		b.Fatalf("write fake qs script: %v", err)
	}

	cliPath := filepath.Join(root, "inir-cli")
	buildCommand := exec.Command("go", "build", "-o", cliPath, "/home/admin/inir-cli")
	buildCommand.Stdout = io.Discard
	buildCommand.Stderr = io.Discard
	if err := buildCommand.Run(); err != nil {
		b.Fatalf("build inir-cli binary: %v", err)
	}

	env := append(os.Environ(),
		"INIR_RUNTIME_DIR="+runtimeDir,
		"INIR_QS_BIN="+qsScript,
		"PATH="+binDir+":"+os.Getenv("PATH"),
	)

	return ipcBenchmarkEnv{cliPath: cliPath, runtimeDir: runtimeDir, env: env}
}
