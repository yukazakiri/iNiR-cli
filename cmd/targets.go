package cmd

import (
	"sort"

	"github.com/yukazakiri/inir-cli/internal/target"
)

func allRegisteredTargets() []string {
	names := target.ListTargets()
	sort.Strings(names)
	return names
}
