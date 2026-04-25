package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

const (
	upstreamGenerateScript     = "/home/admin/inir/scripts/colors/generate_colors_material.py"
	upstreamApplyTargetsScript = "/home/admin/inir/scripts/colors/apply-targets.sh"
)

var (
	benchCLIBinOnce sync.Once
	benchCLIBinPath string
	benchCLIBinErr  error
)

type benchEnv struct {
	home      string
	configDir string
	stateDir  string
	cacheDir  string
	outputDir string

	configFile string
	baseEnv    []string
}

func BenchmarkColorGenerateInirCLI(b *testing.B) {
	cliPath := mustGetBenchCLIPath(b)
	env := mustPrepareBenchEnv(b)

	preflight := runCommand(exec.Command(cliPath,
		"generate",
		"--color", "#FF6B35",
		"--mode", "dark",
		"--scheme", "scheme-tonal-spot",
		"--output", filepath.Join(env.stateDir, "warmup-output"),
	))
	if preflight != nil {
		b.Skipf("inir-cli generate preflight failed: %v", preflight)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		outDir := filepath.Join(env.stateDir, fmt.Sprintf("go-generate-%d", iteration))
		command := exec.Command(cliPath,
			"generate",
			"--color", "#FF6B35",
			"--mode", "dark",
			"--scheme", "scheme-tonal-spot",
			"--output", outDir,
		)
		command.Env = env.baseEnv
		if err := runCommand(command); err != nil {
			b.Fatalf("inir-cli generate failed: %v", err)
		}
	}
}

func BenchmarkColorGenerateDefaultInir(b *testing.B) {
	if _, err := os.Stat(upstreamGenerateScript); err != nil {
		b.Skipf("upstream generate script unavailable: %v", err)
	}

	env := mustPrepareBenchEnv(b)
	preflightOut := filepath.Join(env.stateDir, "upstream-warmup")
	if err := os.MkdirAll(preflightOut, 0755); err != nil {
		b.Fatalf("create warmup output dir: %v", err)
	}
	preflight := exec.Command("python3", upstreamGenerateScript,
		"--color", "#FF6B35",
		"--mode", "dark",
		"--scheme", "vibrant",
		"--json-output", filepath.Join(preflightOut, "colors.json"),
		"--palette-output", filepath.Join(preflightOut, "palette.json"),
		"--terminal-output", filepath.Join(preflightOut, "terminal.json"),
		"--meta-output", filepath.Join(preflightOut, "theme-meta.json"),
	)
	preflight.Env = env.baseEnv
	if err := runCommand(preflight); err != nil {
		b.Skipf("upstream python generate preflight failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		outDir := filepath.Join(env.stateDir, fmt.Sprintf("upstream-generate-%d", iteration))
		if err := os.MkdirAll(outDir, 0755); err != nil {
			b.Fatalf("create output dir: %v", err)
		}
		command := exec.Command("python3", upstreamGenerateScript,
			"--color", "#FF6B35",
			"--mode", "dark",
			"--scheme", "vibrant",
			"--json-output", filepath.Join(outDir, "colors.json"),
			"--palette-output", filepath.Join(outDir, "palette.json"),
			"--terminal-output", filepath.Join(outDir, "terminal.json"),
			"--meta-output", filepath.Join(outDir, "theme-meta.json"),
		)
		command.Env = env.baseEnv
		if err := runCommand(command); err != nil {
			b.Fatalf("upstream generate failed: %v", err)
		}
	}
}

func BenchmarkColorApplyTerminalsInirCLI(b *testing.B) {
	cliPath := mustGetBenchCLIPath(b)
	env := mustPrepareBenchEnv(b)
	mustPrepareGeneratedContract(b, cliPath, env)

	preflight := exec.Command(cliPath, "theme", "apply", "terminals")
	preflight.Env = env.baseEnv
	if err := runCommand(preflight); err != nil {
		b.Skipf("inir-cli theme apply preflight failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := exec.Command(cliPath, "theme", "apply", "terminals")
		command.Env = env.baseEnv
		if err := runCommand(command); err != nil {
			b.Fatalf("inir-cli theme apply terminals failed: %v", err)
		}
	}
}

func BenchmarkColorApplyTerminalsDefaultInir(b *testing.B) {
	if _, err := os.Stat(upstreamApplyTargetsScript); err != nil {
		b.Skipf("upstream apply script unavailable: %v", err)
	}

	cliPath := mustGetBenchCLIPath(b)
	env := mustPrepareBenchEnv(b)
	mustPrepareGeneratedContract(b, cliPath, env)

	preflight := exec.Command("bash", upstreamApplyTargetsScript, "terminals")
	preflight.Env = env.baseEnv
	if err := runCommand(preflight); err != nil {
		b.Skipf("upstream apply preflight failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := exec.Command("bash", upstreamApplyTargetsScript, "terminals")
		command.Env = env.baseEnv
		if err := runCommand(command); err != nil {
			b.Fatalf("upstream apply terminals failed: %v", err)
		}
	}
}

func BenchmarkColorApplyAllInirCLI(b *testing.B) {
	cliPath := mustGetBenchCLIPath(b)
	env := mustPrepareBenchEnv(b)
	mustPrepareGeneratedContract(b, cliPath, env)

	preflight := exec.Command(cliPath, "theme", "apply", "all")
	preflight.Env = env.baseEnv
	if err := runCommand(preflight); err != nil {
		b.Skipf("inir-cli theme apply all preflight failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := exec.Command(cliPath, "theme", "apply", "all")
		command.Env = env.baseEnv
		if err := runCommand(command); err != nil {
			b.Fatalf("inir-cli theme apply all failed: %v", err)
		}
	}
}

func BenchmarkColorApplyAllDefaultInir(b *testing.B) {
	if _, err := os.Stat(upstreamApplyTargetsScript); err != nil {
		b.Skipf("upstream apply script unavailable: %v", err)
	}

	cliPath := mustGetBenchCLIPath(b)
	env := mustPrepareBenchEnv(b)
	mustPrepareGeneratedContract(b, cliPath, env)

	preflight := exec.Command("bash", upstreamApplyTargetsScript, "all")
	preflight.Env = env.baseEnv
	if err := runCommand(preflight); err != nil {
		b.Skipf("upstream apply all preflight failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		command := exec.Command("bash", upstreamApplyTargetsScript, "all")
		command.Env = env.baseEnv
		if err := runCommand(command); err != nil {
			b.Fatalf("upstream apply all failed: %v", err)
		}
	}
}

func mustPrepareGeneratedContract(b *testing.B, cliPath string, env benchEnv) {
	b.Helper()

	command := exec.Command(cliPath,
		"generate",
		"--color", "#FF6B35",
		"--mode", "dark",
		"--scheme", "scheme-tonal-spot",
		"--output", env.outputDir,
	)
	command.Env = env.baseEnv
	if err := runCommand(command); err != nil {
		b.Fatalf("prepare generated contract failed: %v", err)
	}
}

func mustPrepareBenchEnv(b *testing.B) benchEnv {
	b.Helper()

	root := b.TempDir()
	env := benchEnv{
		home:       filepath.Join(root, "home"),
		configDir:  filepath.Join(root, "config"),
		stateDir:   filepath.Join(root, "state"),
		cacheDir:   filepath.Join(root, "cache"),
		outputDir:  filepath.Join(root, "state", "quickshell", "user", "generated"),
		configFile: filepath.Join(root, "config", "inir", "config.json"),
	}

	for _, dir := range []string{env.home, env.configDir, env.stateDir, env.cacheDir, filepath.Dir(env.configFile), env.outputDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			b.Fatalf("create bench dir %s: %v", dir, err)
		}
	}

	if err := writeBenchConfig(env.configFile); err != nil {
		b.Fatalf("write bench config: %v", err)
	}

	env.baseEnv = append(os.Environ(),
		"HOME="+env.home,
		"XDG_CONFIG_HOME="+env.configDir,
		"XDG_STATE_HOME="+env.stateDir,
		"XDG_CACHE_HOME="+env.cacheDir,
	)
	return env
}

func writeBenchConfig(path string) error {
	config := map[string]any{
		"appearance": map[string]any{
			"palette": map[string]any{
				"type": "scheme-tonal-spot",
			},
			"wallpaperTheming": map[string]any{
				"enableAppsAndShell": true,
				"enableTerminal":     true,
				"enableQtApps":       true,
				"terminals": map[string]bool{
					"kitty": true, "alacritty": true, "wezterm": true, "foot": true, "ghostty": true, "konsole": true,
				},
			},
		},
	}
	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0644)
}

func mustGetBenchCLIPath(b *testing.B) string {
	b.Helper()

	benchCLIBinOnce.Do(func() {
		output := filepath.Join(os.TempDir(), "inir-cli-bench-bin")
		command := exec.Command("go", "build", "-o", output, "/home/admin/inir-cli")
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if err := command.Run(); err != nil {
			benchCLIBinErr = err
			return
		}
		benchCLIBinPath = output
	})

	if benchCLIBinErr != nil {
		b.Fatalf("build benchmark cli binary: %v", benchCLIBinErr)
	}
	return benchCLIBinPath
}

func runCommand(command *exec.Cmd) error {
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Run()
}
