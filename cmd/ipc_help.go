// File: ipc_help.go
//
// Per-target help formatting for IPC commands.
// Prints target name, description, available functions with args,
// family, and example keybinds.
package cmd

import (
	"fmt"
	"io"
	"strings"
)

func printIPCTargetHelp(writer io.Writer, meta ipcTarget) {
	fmt.Fprintf(writer, "%s — %s\n\n", meta.Name, meta.Description)
	fmt.Fprintln(writer, "Available functions:")
	for _, fn := range meta.Functions {
		displayName := strings.TrimSpace(fn.Name + " " + fn.Args)
		if fn.Description == "" {
			fmt.Fprintf(writer, "  %s\n", displayName)
			continue
		}
		fmt.Fprintf(writer, "  %-28s %s\n", displayName, fn.Description)
	}
	fmt.Fprintf(writer, "\nFamily: %s\n", meta.Family)
	if meta.Example != "" {
		fmt.Fprintf(writer, "\nExample keybind:\n  %s\n", meta.Example)
	}
}
