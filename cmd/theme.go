package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yukazakiri/inir-cli/internal/color"
	"github.com/yukazakiri/inir-cli/internal/config"
	"github.com/yukazakiri/inir-cli/internal/target"
	"github.com/yukazakiri/inir-cli/internal/template"
)

var (
	flagImage              string
	flagColor              string
	flagMode               string
	flagScheme             string
	flagConfig             string
	flagTemplateDir        string
	flagOutputDir          string
	flagCache              string
	flagHarmony            float64
	flagTermSaturation     float64
	flagTermBrightness     float64
	flagTermBgBrightness   float64
	flagHarmonizeThreshold float64
	flagTermFgBoost        float64
	flagColorStrength      float64
	flagSoften             bool
	flagBlendBgFg          bool
	flagSkipConfigWrite    bool
	flagForceDarkTerminal  bool
)

func init() {
	generateCmd.Flags().StringVar(&flagImage, "image", "", "Path to wallpaper image")
	generateCmd.Flags().StringVar(&flagColor, "color", "", "Hex color seed (e.g. #FF6B35)")
	generateCmd.Flags().StringVar(&flagMode, "mode", "", "dark or light (auto-detect if empty)")
	generateCmd.Flags().StringVar(&flagScheme, "scheme", "auto", "Material You scheme variant")
	generateCmd.Flags().StringVar(&flagConfig, "config", "", "Path to config.json")
	generateCmd.Flags().StringVar(&flagTemplateDir, "templates", "", "Template directory (matugen/)")
	generateCmd.Flags().StringVar(&flagOutputDir, "output", "", "Output directory for generated files")
	generateCmd.Flags().StringVar(&flagCache, "cache", "", "Path to store seed color cache")
	generateCmd.Flags().Float64Var(&flagHarmony, "harmony", 0.4, "Color hue shift towards accent (0-1)")
	generateCmd.Flags().Float64Var(&flagTermSaturation, "term-saturation", 0.65, "Terminal color saturation (0-1)")
	generateCmd.Flags().Float64Var(&flagTermBrightness, "term-brightness", 0.60, "Terminal color brightness (0-1)")
	generateCmd.Flags().Float64Var(&flagTermBgBrightness, "term-bg-brightness", 0.50, "Terminal background brightness (0-1)")
	generateCmd.Flags().Float64Var(&flagHarmonizeThreshold, "harmonize-threshold", 100, "Max threshold angle for hue shift")
	generateCmd.Flags().Float64Var(&flagTermFgBoost, "term-fg-boost", 0.35, "Terminal foreground boost")
	generateCmd.Flags().Float64Var(&flagColorStrength, "color-strength", 1.0, "Chroma multiplier for accents")
	generateCmd.Flags().BoolVar(&flagSoften, "soften", false, "Soften generated colors")
	generateCmd.Flags().BoolVar(&flagBlendBgFg, "blend-bg-fg", false, "Shift terminal bg/fg towards accent")
	generateCmd.Flags().BoolVar(&flagSkipConfigWrite, "skip-config-write", false, "Don't update config.json wallpaper path")
	generateCmd.Flags().BoolVar(&flagForceDarkTerminal, "force-dark-terminal", false, "Force dark mode for terminal colors")

	themeCmd.AddCommand(themeApplyCmd)
	themeCmd.AddCommand(themeGenerateCmd)
}

var themeApplyCmd = &cobra.Command{
	Use:   "apply [targets...]",
	Short: "Generate colors and apply to specified targets",
	Long:  `Full pipeline: generate Material You colors from wallpaper, then apply to specified targets.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runThemeApply,
}

var themeGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate color palette (full pipeline)",
	RunE:  runThemeGenerate,
}

func resolveXDG() (configHome, stateHome, cacheHome string) {
	configHome = os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	stateHome = os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(os.Getenv("HOME"), ".local/state")
	}
	cacheHome = os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		cacheHome = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return
}

func loadConfig(configPath string) *config.Config {
	if configPath == "" {
		configHome, _, _ := resolveXDG()
		configPath = filepath.Join(configHome, "inir", "config.json")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fallback := config.DefaultConfig()
		applyWallpaperThemingFromRawConfig(configPath, fallback)
		return fallback
	}
	return cfg
}

func applyWallpaperThemingFromRawConfig(configPath string, cfg *config.Config) {
	if cfg == nil {
		return
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}

	appearance, _ := root["appearance"].(map[string]interface{})
	wallpaperTheming, _ := appearance["wallpaperTheming"].(map[string]interface{})
	if len(wallpaperTheming) == 0 {
		if direct, ok := root["wallpaperTheming"].(map[string]interface{}); ok {
			wallpaperTheming = direct
		}
	}

	readBool := func(key string, current bool) bool {
		value, ok := wallpaperTheming[key]
		if !ok {
			return current
		}
		b, ok := value.(bool)
		if !ok {
			return current
		}
		return b
	}

	cfg.WallpaperTheming.EnableAppsAndShell = readBool("enableAppsAndShell", cfg.WallpaperTheming.EnableAppsAndShell)
	cfg.WallpaperTheming.EnableTerminal = readBool("enableTerminal", cfg.WallpaperTheming.EnableTerminal)
	cfg.WallpaperTheming.EnableQtApps = readBool("enableQtApps", cfg.WallpaperTheming.EnableQtApps)
	cfg.WallpaperTheming.EnableVesktop = readBool("enableVesktop", cfg.WallpaperTheming.EnableVesktop)
	cfg.WallpaperTheming.EnableZed = readBool("enableZed", cfg.WallpaperTheming.EnableZed)
	cfg.WallpaperTheming.EnableVSCode = readBool("enableVSCode", cfg.WallpaperTheming.EnableVSCode)
	cfg.WallpaperTheming.EnableChrome = readBool("enableChrome", cfg.WallpaperTheming.EnableChrome)
	cfg.WallpaperTheming.EnableSpicetify = readBool("enableSpicetify", cfg.WallpaperTheming.EnableSpicetify)
	cfg.WallpaperTheming.EnableAdwSteam = readBool("enableAdwSteam", cfg.WallpaperTheming.EnableAdwSteam)
	cfg.WallpaperTheming.EnablePearDesktop = readBool("enablePearDesktop", cfg.WallpaperTheming.EnablePearDesktop)
	cfg.WallpaperTheming.EnableNeovim = readBool("enableNeovim", cfg.WallpaperTheming.EnableNeovim)
	cfg.WallpaperTheming.EnableOpenCode = readBool("enableOpenCode", cfg.WallpaperTheming.EnableOpenCode)
}

func resolveOutputDir() string {
	if flagOutputDir != "" {
		return flagOutputDir
	}
	_, stateHome, _ := resolveXDG()
	return filepath.Join(stateHome, "quickshell", "user", "generated")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	cfg := loadConfig(flagConfig)
	outputDir := resolveOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	scheme := flagScheme
	imagePath := flagImage
	seedColor := flagColor
	mode := flagMode

	if scheme == "" || scheme == "auto" {
		scheme = cfg.Appearance.Palette.Type
	}
	if scheme == "" || scheme == "auto" {
		if imagePath != "" {
			detected, err := detectScheme(imagePath)
			if err == nil {
				scheme = detected
			} else {
				scheme = "scheme-tonal-spot"
			}
		} else {
			scheme = "scheme-tonal-spot"
		}
	}

	if mode == "" {
		mode = "dark"
	}

	if seedColor == "" && imagePath == "" {
		return fmt.Errorf("either --image or --color is required")
	}

	harmony := flagHarmony
	if harmony == 0.4 && cfg.TerminalColorAdjustments.Harmony != 0 {
		harmony = cfg.TerminalColorAdjustments.Harmony
	}
	termSat := flagTermSaturation
	if termSat == 0.65 && cfg.TerminalColorAdjustments.Saturation != 0 {
		termSat = cfg.TerminalColorAdjustments.Saturation
	}
	termBri := flagTermBrightness
	if termBri == 0.60 && cfg.TerminalColorAdjustments.Brightness != 0 {
		termBri = cfg.TerminalColorAdjustments.Brightness
	}
	termBgBri := flagTermBgBrightness
	if termBgBri == 0.50 && cfg.TerminalColorAdjustments.BackgroundBrightness != 0 {
		termBgBri = cfg.TerminalColorAdjustments.BackgroundBrightness
	}
	colorStrength := flagColorStrength
	if colorStrength == 1.0 && cfg.WallpaperTheming.ColorStrength != 0 {
		colorStrength = cfg.WallpaperTheming.ColorStrength
	}

	genOpts := color.GenerateOptions{
		ImagePath:          imagePath,
		SeedColor:          seedColor,
		Mode:               mode,
		Scheme:             scheme,
		Harmony:            harmony,
		TermSaturation:     termSat,
		TermBrightness:     termBri,
		TermBgBrightness:   termBgBri,
		HarmonizeThreshold: flagHarmonizeThreshold,
		TermFgBoost:        flagTermFgBoost,
		ColorStrength:      colorStrength,
		Soften:             flagSoften || cfg.SoftenColors,
		BlendBgFg:          flagBlendBgFg,
	}

	result, err := color.Generate(genOpts)
	if err != nil {
		return fmt.Errorf("color generation failed: %w", err)
	}

	if flagCache != "" {
		os.WriteFile(flagCache, []byte(result.SeedColor), 0644)
	}

	colorsPath := filepath.Join(outputDir, "colors.json")
	palettePath := filepath.Join(outputDir, "palette.json")
	terminalPath := filepath.Join(outputDir, "terminal.json")
	metaPath := filepath.Join(outputDir, "theme-meta.json")
	scssPath := filepath.Join(outputDir, "material_colors.scss")

	if err := result.WriteJSON(colorsPath, palettePath, terminalPath, metaPath); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}

	if err := result.WriteSCSS(scssPath); err != nil {
		fmt.Fprintf(os.Stderr, "[inir-cli] Warning: SCSS write failed: %v\n", err)
	}

	if flagForceDarkTerminal {
		darkOpts := genOpts
		darkOpts.Mode = "dark"
		darkResult, err := color.Generate(darkOpts)
		if err == nil {
			darkResult.WriteTerminalJSON(terminalPath)
			darkResult.WriteSCSS(scssPath)
			*result = *darkResult
		}
	}

	if err := writeChromiumThemeContracts(outputDir, result.Palette); err != nil {
		fmt.Fprintf(os.Stderr, "[inir-cli] Warning: chromium.theme contract write failed: %v\n", err)
	}

	if flagTemplateDir != "" {
		if err := template.RenderAll(flagTemplateDir, result); err != nil {
			fmt.Fprintf(os.Stderr, "[inir-cli] Warning: template rendering failed: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "[inir-cli] Generated colors from seed %s (scheme=%s, mode=%s)\n", result.SeedColor, scheme, mode)
	return nil
}

func runThemeGenerate(cmd *cobra.Command, args []string) error {
	return runGenerate(generateCmd, args)
}

func runThemeApply(cmd *cobra.Command, args []string) error {
	cfg := loadConfig(flagConfig)
	outputDir := resolveOutputDir()

	colorsPath := filepath.Join(outputDir, "colors.json")
	palettePath := filepath.Join(outputDir, "palette.json")
	terminalPath := filepath.Join(outputDir, "terminal.json")
	scssPath := filepath.Join(outputDir, "material_colors.scss")
	metaPath := filepath.Join(outputDir, "theme-meta.json")

	if _, err := os.Stat(colorsPath); os.IsNotExist(err) {
		return fmt.Errorf("no colors.json found — run 'generate' first")
	}

	targetList := args
	if len(targetList) == 1 && targetList[0] == "all" {
		targetList = allRegisteredTargets()
	}

	ctx := &target.Context{
		Config:       cfg,
		ColorsPath:   colorsPath,
		PalettePath:  palettePath,
		TerminalPath: terminalPath,
		SCSSPath:     scssPath,
		MetaPath:     metaPath,
		OutputDir:    outputDir,
	}

	for _, t := range targetList {
		name := strings.TrimSpace(t)
		applier := target.GetApplier(name)
		if applier == nil {
			fmt.Fprintf(os.Stderr, "[inir-cli] Unknown target: %s\n", name)
			continue
		}
		if err := applier.Apply(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[inir-cli] Target %s failed: %v\n", name, err)
		}
	}

	return nil
}

func detectScheme(imagePath string) (string, error) {
	return color.DetectSchemeFromImage(imagePath)
}
