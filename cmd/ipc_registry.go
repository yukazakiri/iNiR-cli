// File: ipc_registry.go
//
// IPC target and function types, plus lookup helpers.
//
// The registry is assembled at init time by merging generated data
// (ipc_registry_generated.go) with manual overrides (ipc_registry_overrides.go).
//
// Types:
//   - ipcTarget:    A shell IPC target (name, description, family, functions, example)
//   - ipcFunction:  A callable function on a target (name, args, description)
//
// Helpers:
//   - findIPCTarget:       Look up a target by exact name
//   - ipcAliasesForTarget: Get kebab-case aliases for a canonical target name
package cmd

import "sort"

type ipcFunction struct {
	Name        string
	Args        string
	Description string
}

type ipcTarget struct {
	Name        string
	Description string
	Family      string
	Functions   []ipcFunction
	Example     string
}

var ipcKebabAliases = buildIPCAliases()

var ipcTargets = buildIPCTargets()

func findIPCTarget(name string) (ipcTarget, bool) {
	for _, meta := range ipcTargets {
		if meta.Name == name {
			return meta, true
		}
	}
	return ipcTarget{}, false
}

func ipcAliasesForTarget(targetName string) []string {
	aliases := make([]string, 0, 1)
	for alias, canonical := range ipcKebabAliases {
		if canonical == targetName {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}
