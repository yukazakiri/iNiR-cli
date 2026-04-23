package cmd

import "testing"

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
}
