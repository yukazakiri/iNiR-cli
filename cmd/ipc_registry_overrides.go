// File: ipc_registry_overrides.go
//
// Manual override layer for the IPC registry.
//
// To add or modify a target without touching generated code:
//   1. Add an entry to ipcTargetOverrides (key = target name)
//   2. Add kebab-case aliases to ipcAliasOverrides if needed
//
// The buildIPCTargets() and buildIPCAliases() functions merge
// overrides on top of the generated data at init time.
package cmd

var ipcAliasOverrides = map[string]string{}

var ipcTargetOverrides = map[string]ipcTarget{}

func buildIPCAliases() map[string]string {
	aliases := cloneAliasMap(generatedIPCKebabAliases)
	for alias, canonical := range ipcAliasOverrides {
		aliases[alias] = canonical
	}
	return aliases
}

func buildIPCTargets() []ipcTarget {
	overrides := ipcTargetOverrides
	if len(overrides) == 0 {
		return cloneTargets(generatedIPCTargets)
	}

	merged := cloneTargets(generatedIPCTargets)
	for index, meta := range merged {
		if override, ok := overrides[meta.Name]; ok {
			merged[index] = cloneTarget(override)
		}
	}
	for name, override := range overrides {
		if !containsIPCTarget(merged, name) {
			merged = append(merged, cloneTarget(override))
		}
	}
	return merged
}

func containsIPCTarget(targets []ipcTarget, name string) bool {
	for _, meta := range targets {
		if meta.Name == name {
			return true
		}
	}
	return false
}

func cloneTargets(input []ipcTarget) []ipcTarget {
	cloned := make([]ipcTarget, 0, len(input))
	for _, meta := range input {
		cloned = append(cloned, cloneTarget(meta))
	}
	return cloned
}

func cloneTarget(meta ipcTarget) ipcTarget {
	copyMeta := meta
	copyMeta.Functions = append([]ipcFunction(nil), meta.Functions...)
	return copyMeta
}

func cloneAliasMap(input map[string]string) map[string]string {
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
