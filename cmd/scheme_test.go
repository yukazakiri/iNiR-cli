package cmd

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/yukazakiri/inir-cli/internal/presets"
)

func TestAllSchemeTargetsIncludesZedAndNoDuplicates(t *testing.T) {
	t.Parallel()

	targets := allSchemeTargets()
	seen := map[string]bool{}
	foundZed := false

	for _, target := range targets {
		if seen[target] {
			t.Fatalf("duplicate target in scheme list: %s", target)
		}
		seen[target] = true
		if target == "zed" {
			foundZed = true
		}
	}

	if !foundZed {
		t.Fatalf("expected zed target in scheme apply-all list")
	}

	if !sort.StringsAreSorted(targets) {
		t.Fatalf("expected scheme targets to be sorted for deterministic all-target execution")
	}
}

func TestResolveSchemePresetByName(t *testing.T) {
	t.Parallel()

	preset, err := resolveSchemePreset("catppuccin-mocha", false, nil)
	if err != nil {
		t.Fatalf("resolveSchemePreset returned error: %v", err)
	}
	if preset == nil || preset.ID != "catppuccin-mocha" {
		t.Fatalf("expected named preset catppuccin-mocha, got %+v", preset)
	}
}

func TestResolveSchemePresetRandom(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(1))
	preset, err := resolveSchemePreset("", true, rng)
	if err != nil {
		t.Fatalf("resolveSchemePreset(random) returned error: %v", err)
	}
	if preset == nil {
		t.Fatalf("expected random preset, got nil")
	}

	found := false
	for _, p := range presets.Presets {
		if p.ID == preset.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("random preset %q not found in preset catalog", preset.ID)
	}
}
