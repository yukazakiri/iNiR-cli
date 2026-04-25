// Package cmd wires all cobra CLI commands and orchestrates the theme pipeline.
//
// The command tree:
//
//	inir-cli generate          — Generate palette from wallpaper/seed color
//	inir-cli apply [targets]   — Redirect to theme apply
//	inir-cli scheme [name]     — Apply a built-in preset theme
//	inir-cli auto-detect [img] — Detect best scheme variant
//	inir-cli theme generate    — Generate (namespaced)
//	inir-cli theme apply [t]   — Apply generated colors to targets
//	inir-cli theme list-targets — List built-in + external targets
//	inir-cli theme scaffold-target <id> — Create external target spec
//	inir-cli ipc <t> <fn>      — Raw Quickshell IPC passthrough
//	inir-cli <target> <fn>     — Upstream-style IPC (e.g. overview toggle)
//
// Heavy computation lives in internal/; this package handles flags, config,
// XDG resolution, and pipeline orchestration only.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "inir-cli",
	Short: "Wallpaper-based Material You theme generator and applier",
	Long: `inir-cli generates Material You color palettes from wallpapers or seed colors
and applies them across 30+ targets including GTK, KDE, terminals, editors,
browsers, music players, and more.

It acts as the backbone of the iNiR desktop shell, usable standalone or
orchestrated by Quickshell.`,
}

func Execute() error {
	return rootCmd.Execute()
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate color palette from wallpaper or seed color",
	RunE:  runGenerate,
}

var applyCmd = &cobra.Command{
	Use:   "apply [targets...]",
	Short: "Apply generated colors to specified targets",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runApply,
}

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Theme workflow commands (generate, apply, list-targets, scaffold-target)",
}

var autoDetectCmd = &cobra.Command{
	Use:   "auto-detect [image]",
	Short: "Detect the best Material You scheme variant for an image",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoDetect,
}

func runApply(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("use 'inir-cli theme apply <targets...>' instead")
}

func runAutoDetect(cmd *cobra.Command, args []string) error {
	scheme, err := detectScheme(args[0])
	if err != nil {
		return err
	}
	fmt.Println(scheme)
	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(themeCmd)
	rootCmd.AddCommand(autoDetectCmd)
}
