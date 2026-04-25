package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yukazakiri/inir-cli/internal/config"
	targetpkg "github.com/yukazakiri/inir-cli/internal/target"
)

type targetFailure struct {
	Target string
	Error  string
}

func applyThemeTargets(cfg *config.Config, configPath string, contract outputContract, requestedTargets []string) error {
	externalTargets, err := discoverExternalTargets(configPath)
	if err != nil {
		return err
	}

	resolvedTargets := resolveRequestedTargets(requestedTargets, externalTargets)
	ctx := &targetpkg.Context{
		Config:       cfg,
		ColorsPath:   contract.ColorsPath,
		PalettePath:  contract.PalettePath,
		TerminalPath: contract.TerminalPath,
		SCSSPath:     contract.SCSSPath,
		MetaPath:     contract.MetaPath,
		OutputDir:    contract.OutputDir,
	}

	failures := make([]targetFailure, 0)
	for _, id := range resolvedTargets {
		if builtin := targetpkg.GetApplier(id); builtin != nil {
			if err := builtin.Apply(ctx); err != nil {
				failures = append(failures, targetFailure{Target: id, Error: err.Error()})
			}
			continue
		}

		spec, ok := externalTargets[id]
		if !ok {
			failures = append(failures, targetFailure{Target: id, Error: "target not found"})
			continue
		}

		if err := runExternalTarget(spec, contract, configPath); err != nil {
			failures = append(failures, targetFailure{Target: id, Error: err.Error()})
		}
	}

	if len(failures) == 0 {
		return nil
	}

	notifyApplyFailures(failures)
	return formatApplyFailures(failures)
}

func resolveRequestedTargets(requested []string, external map[string]externalThemeTarget) []string {
	if len(requested) == 1 && strings.TrimSpace(requested[0]) == "all" {
		return listAllTargets(external)
	}

	targets := make([]string, 0, len(requested))
	for _, targetName := range requested {
		targetName = strings.TrimSpace(targetName)
		if targetName == "" {
			continue
		}
		targets = append(targets, targetName)
	}
	return targets
}

func listAllTargets(external map[string]externalThemeTarget) []string {
	builtin := allRegisteredTargets()
	all := append([]string{}, builtin...)
	all = append(all, listExternalTargetIDs(external)...)
	return uniqueSortedTargets(all)
}

func uniqueSortedTargets(targets []string) []string {
	seen := make(map[string]struct{}, len(targets))
	unique := make([]string, 0, len(targets))
	for _, targetName := range targets {
		if _, ok := seen[targetName]; ok {
			continue
		}
		seen[targetName] = struct{}{}
		unique = append(unique, targetName)
	}
	sort.Strings(unique)
	return unique
}

func formatApplyFailures(failures []targetFailure) error {
	if len(failures) == 0 {
		return nil
	}
	parts := make([]string, 0, len(failures))
	for _, failure := range failures {
		parts = append(parts, fmt.Sprintf("%s: %s", failure.Target, failure.Error))
	}
	return fmt.Errorf("theme apply failed for %d target(s): %s", len(failures), strings.Join(parts, " | "))
}

func defaultConfigPath() string {
	configHome, _, _ := resolveXDG()
	return filepath.Join(configHome, "inir", "config.json")
}
