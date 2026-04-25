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
