// File: ipc_parse.go
//
// Argument parsing and validation for IPC commands.
//
//   - parseIPCPrefixArgs:   Extracts -c/--config and --help from args, returns rest
//   - normalizeIPCTarget:   Resolves kebab-case aliases to canonical camelCase names
//   - validateIPCFunction:  Checks that a function name exists for a given target
//   - firstSentence:         Truncates description text at the first period
package cmd

import (
	"fmt"
	"strings"
)

type ipcPrefixArgs struct {
	Config string
	Rest   []string
	Help   bool
}

func parseIPCPrefixArgs(args []string) (ipcPrefixArgs, error) {
	parsed := ipcPrefixArgs{Rest: args}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "-c", "--config":
			if index+1 >= len(args) {
				return parsed, fmt.Errorf("missing path after %s", args[index])
			}
			parsed.Config = args[index+1]
			index++
		case "--":
			parsed.Rest = args[index+1:]
			return parsed, nil
		case "-h", "--help":
			parsed.Help = true
			parsed.Rest = nil
			return parsed, nil
		default:
			parsed.Rest = args[index:]
			return parsed, nil
		}
	}
	parsed.Rest = nil
	return parsed, nil
}

func normalizeIPCTarget(input string) (string, bool) {
	if canonical, ok := ipcKebabAliases[input]; ok {
		return canonical, true
	}
	if _, ok := findIPCTarget(input); ok {
		return input, true
	}
	return input, false
}

func validateIPCFunction(meta ipcTarget, functionName string) error {
	for _, fn := range meta.Functions {
		if fn.Name == functionName {
			return nil
		}
	}
	return fmt.Errorf("unknown function %q for IPC target %q", functionName, meta.Name)
}

func firstSentence(text string) string {
	if before, _, found := strings.Cut(text, "."); found {
		return before + "."
	}
	return text
}
