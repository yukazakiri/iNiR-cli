// File: targets.go
//
// Helper that returns all registered built-in target names, sorted alphabetically.
// Used by "theme apply all" and "theme list-targets" to enumerate available targets.
package cmd

import (
	"sort"

	targetpkg "github.com/yukazakiri/inir-cli/internal/target"
)

func allRegisteredTargets() []string {
	names := targetpkg.ListTargets()
	sort.Strings(names)
	return names
}
