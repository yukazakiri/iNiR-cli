// File: theme_pipeline_targets.go
//
// External target discovery, validation, and execution.
//
// Discovery searches these directories in order:
//   1. INIR_THEME_TARGETS_DIR (colon-separated, from env)
//   2. $(dirname config.json)/targets/
//   3. ~/.config/inir/targets/
//   4. ~/.config/inir-cli/targets/
//
// Each .json file defines an external target with id, command, args, inputs, and env.
// The "command" type is the only supported type currently.
// External targets receive contract paths via environment variables (INIR_OUTPUT_DIR, etc.).
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var externalTargetCommandRunner = runExternalTargetCommand

var validTargetIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type externalThemeTarget struct {
	ID          string            `json:"id"`
	Label       string            `json:"label,omitempty"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type,omitempty"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Inputs      []string          `json:"inputs,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
}

func discoverExternalTargets(configPath string) (map[string]externalThemeTarget, error) {
	targetDirs := resolveExternalTargetDirs(configPath)
	targets := make(map[string]externalThemeTarget)

	for _, dir := range targetDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read target directory %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			specPath := filepath.Join(dir, entry.Name())
			spec, err := loadExternalTargetSpec(specPath)
			if err != nil {
				return nil, err
			}
			if spec.Enabled != nil && !*spec.Enabled {
				continue
			}
			if _, exists := targets[spec.ID]; exists {
				return nil, fmt.Errorf("duplicate external target id %q", spec.ID)
			}
			targets[spec.ID] = spec
		}
	}

	return targets, nil
}

func loadExternalTargetSpec(path string) (externalThemeTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return externalThemeTarget{}, fmt.Errorf("read target spec %s: %w", path, err)
	}

	var spec externalThemeTarget
	if err := json.Unmarshal(data, &spec); err != nil {
		return externalThemeTarget{}, fmt.Errorf("parse target spec %s: %w", path, err)
	}

	spec.ID = strings.TrimSpace(spec.ID)
	if !validTargetIDPattern.MatchString(spec.ID) {
		return externalThemeTarget{}, fmt.Errorf("invalid target id in %s: %q", path, spec.ID)
	}
	spec.Command = strings.TrimSpace(spec.Command)
	if spec.Command == "" {
		return externalThemeTarget{}, fmt.Errorf("missing command in %s", path)
	}
	if spec.Type == "" {
		spec.Type = "command"
	}
	if spec.Type != "command" {
		return externalThemeTarget{}, fmt.Errorf("unsupported target type %q in %s", spec.Type, path)
	}

	return spec, nil
}

func resolveExternalTargetDirs(configPath string) []string {
	dirs := make([]string, 0, 4)

	if fromEnv := os.Getenv("INIR_THEME_TARGETS_DIR"); fromEnv != "" {
		for _, part := range strings.Split(fromEnv, ":") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			dirs = append(dirs, part)
		}
	}

	if configPath != "" {
		dirs = append(dirs, filepath.Join(filepath.Dir(configPath), "targets"))
	}

	configHome, _, _ := resolveXDG()
	dirs = append(dirs,
		filepath.Join(configHome, "inir", "targets"),
		filepath.Join(configHome, "inir-cli", "targets"),
	)

	return dedupePaths(dirs)
}

func listExternalTargetIDs(targets map[string]externalThemeTarget) []string {
	ids := make([]string, 0, len(targets))
	for id := range targets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func runExternalTarget(spec externalThemeTarget, contract outputContract, configPath string) error {
	env := append(os.Environ(),
		"INIR_OUTPUT_DIR="+contract.OutputDir,
		"INIR_COLORS_JSON="+contract.ColorsPath,
		"INIR_PALETTE_JSON="+contract.PalettePath,
		"INIR_TERMINAL_JSON="+contract.TerminalPath,
		"INIR_THEME_META_JSON="+contract.MetaPath,
		"INIR_MATERIAL_SCSS="+contract.SCSSPath,
		"INIR_CONFIG_JSON="+configPath,
	)

	for key, value := range spec.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	if err := externalTargetCommandRunner(spec.Command, spec.Args, env); err != nil {
		return fmt.Errorf("external target %s failed: %w", spec.ID, err)
	}
	return nil
}

func runExternalTargetCommand(command string, args []string, env []string) error {
	cmd := exec.Command(command, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dedupePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		unique = append(unique, clean)
	}
	return unique
}
