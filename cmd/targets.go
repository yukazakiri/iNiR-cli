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
