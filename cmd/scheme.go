package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yukazakiri/inir-cli/internal/presets"
	"github.com/yukazakiri/inir-cli/internal/target"
)

var (
	flagSchemeConfig     string
	flagSchemeOutputDir  string
	flagSchemeList       bool
	flagSchemeApply      bool
	flagSchemeRandom     bool
	flagSchemeHarmony    float64
	flagSchemeTermSat    float64
	flagSchemeTermBri    float64
)

var schemeCmd = &cobra.Command{
	Use:   "scheme [theme-name]",
	Short: "Apply a built-in static theme preset",
	Long: `Apply one of 44+ built-in theme presets (e.g. catppuccin-mocha, tokyo-night, gruvbox-material).

Use 'inir-cli scheme --list' to see all available themes.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScheme,
}

func init() {
	schemeCmd.Flags().StringVar(&flagSchemeConfig, "config", "", "Path to config.json")
	schemeCmd.Flags().StringVar(&flagSchemeOutputDir, "output", "", "Output directory for generated files")
	schemeCmd.Flags().BoolVar(&flagSchemeList, "list", false, "List all available theme presets")
	schemeCmd.Flags().BoolVar(&flagSchemeApply, "apply", false, "Apply theme to all targets after generating")
	schemeCmd.Flags().BoolVar(&flagSchemeRandom, "random", false, "Pick and apply a random preset theme")
	schemeCmd.Flags().Float64Var(&flagSchemeHarmony, "harmony", 0.40, "Terminal color harmony (0-1)")
	schemeCmd.Flags().Float64Var(&flagSchemeTermSat, "term-saturation", 0.65, "Terminal color saturation (0-1)")
	schemeCmd.Flags().Float64Var(&flagSchemeTermBri, "term-brightness", 0.60, "Terminal color brightness (0-1)")

	rootCmd.AddCommand(schemeCmd)
}

func runScheme(cmd *cobra.Command, args []string) error {
	if flagSchemeList {
		return listPresets()
	}

	if len(args) == 0 && !flagSchemeRandom {
		return fmt.Errorf("theme name required (use --list to see available themes)")
	}

	themeName := ""
	if len(args) > 0 {
		themeName = args[0]
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	preset, err := resolveSchemePreset(themeName, flagSchemeRandom, rng)
	if err != nil {
		return err
	}

	outputDir := flagSchemeOutputDir
	if outputDir == "" {
		_, stateHome, _ := resolveXDG()
		outputDir = filepath.Join(stateHome, "quickshell", "user", "generated")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	colorsMap := preset.Colors.ToMap()

	var termColors map[string]string
	if preset.Colors.HasExplicitTerminalColors() {
		termColors = preset.Colors.TerminalColorMap()
	} else {
		termColors = generatePresetTerminalColors(&preset.Colors, flagSchemeHarmony, flagSchemeTermSat, flagSchemeTermBri)
	}

	colorsJSON := make(map[string]string)
	for k, v := range colorsMap {
		colorsJSON[k] = v
	}
	for k, v := range termColors {
		colorsJSON[k] = v
	}

	colorsPath := filepath.Join(outputDir, "colors.json")
	palettePath := filepath.Join(outputDir, "palette.json")
	terminalPath := filepath.Join(outputDir, "terminal.json")
	metaPath := filepath.Join(outputDir, "theme-meta.json")
	scssPath := filepath.Join(outputDir, "material_colors.scss")

	if err := writeSchemeJSON(colorsPath, colorsJSON); err != nil {
		return fmt.Errorf("write colors.json: %w", err)
	}
	if err := writeSchemeJSON(palettePath, colorsMap); err != nil {
		return fmt.Errorf("write palette.json: %w", err)
	}
	if err := writeSchemeJSON(terminalPath, termColors); err != nil {
		return fmt.Errorf("write terminal.json: %w", err)
	}

	meta := map[string]interface{}{
		"source":        "preset",
		"preset":        preset.ID,
		"mode":          modeString(preset.Colors.Darkmode),
		"scheme":        "preset",
		"transparent":   preset.Colors.Transparent,
		"term_source":   termSource(preset.Colors.HasExplicitTerminalColors()),
		"generated_by":  "inir-cli",
	}
	if err := writeSchemeJSON(metaPath, meta); err != nil {
		return fmt.Errorf("write theme-meta.json: %w", err)
	}

	if err := writeSchemeSCSS(scssPath, &preset.Colors, colorsMap, termColors); err != nil {
		fmt.Fprintf(os.Stderr, "[inir-cli] Warning: SCSS write failed: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "[inir-cli] Applied preset theme: %s (%s)\n", preset.Name, preset.ID)
	fmt.Fprintf(os.Stderr, "[inir-cli] Mode: %s, Terminal: %s\n", meta["mode"], meta["term_source"])

	if flagSchemeApply {
		return applySchemeTargets(outputDir)
	}

	return nil
}

func resolveSchemePreset(themeName string, randomPick bool, rng *rand.Rand) (*presets.Preset, error) {
	if randomPick {
		if len(presets.Presets) == 0 {
			return nil, fmt.Errorf("no presets available")
		}
		if rng == nil {
			rng = rand.New(rand.NewSource(time.Now().UnixNano()))
		}
		picked := presets.Presets[rng.Intn(len(presets.Presets))]
		return &picked, nil
	}

	preset := presets.GetPreset(themeName)
	if preset == nil {
		return nil, fmt.Errorf("unknown theme: %q (use --list to see available themes)", themeName)
	}

	return preset, nil
}

func listPresets() error {
	for _, p := range presets.Presets {
		mode := "dark"
		if !p.Colors.Darkmode {
			mode = "light"
		}
		tags := strings.Join(p.Tags, ", ")
		if tags != "" {
			tags = " [" + tags + "]"
		}
		fmt.Printf("  %-28s %s%s (%s)\n", p.ID, p.Name, tags, mode)
	}
	fmt.Fprintf(os.Stderr, "\n%d themes available\n", len(presets.Presets))
	return nil
}

func applySchemeTargets(outputDir string) error {
	colorsPath := filepath.Join(outputDir, "colors.json")
	palettePath := filepath.Join(outputDir, "palette.json")
	terminalPath := filepath.Join(outputDir, "terminal.json")
	scssPath := filepath.Join(outputDir, "material_colors.scss")
	metaPath := filepath.Join(outputDir, "theme-meta.json")

	cfg := loadConfig(flagSchemeConfig)

	ctx := &target.Context{
		Config:       cfg,
		ColorsPath:   colorsPath,
		PalettePath:  palettePath,
		TerminalPath: terminalPath,
		SCSSPath:     scssPath,
		MetaPath:     metaPath,
		OutputDir:    outputDir,
	}

	targets := allSchemeTargets()
	for _, t := range targets {
		applier := target.GetApplier(t)
		if applier == nil {
			continue
		}
		if err := applier.Apply(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[inir-cli] Target %s: %v\n", t, err)
		}
	}
	return nil
}

func allSchemeTargets() []string {
	return allRegisteredTargets()
}

func generatePresetTerminalColors(c *presets.PresetColors, harmony, termSat, termBri float64) map[string]string {
	isDark := c.Darkmode

	bg := c.Background
	if c.SurfaceContainerLow != "" {
		bg = c.SurfaceContainerLow
	}

	primaryR, primaryG, primaryB := hexToRGB(c.Primary)
	primaryH, primaryS, _ := rgbToHSL(primaryR, primaryG, primaryB)

	normalLight := 0.42 + termBri*0.35
	brightLight := 0.55 + termBri*0.30
	if !isDark {
		normalLight = 0.58 - termBri*0.30
		brightLight = 0.48 - termBri*0.30
	}
	normalSat := math.Min(0.85, termSat*1.3)
	brightSat := math.Min(0.90, termSat*1.3+0.05)

	harmonize := func(targetHue, sat, light float64) string {
		finalHue := targetHue
		if primaryS > 0.08 && harmony > 0 {
			hueDiff := primaryH - targetHue
			if hueDiff > 0.5 {
				hueDiff -= 1
			}
			if hueDiff < -0.5 {
				hueDiff += 1
			}
			maxShift := 0.033
			rawShift := hueDiff * harmony * 0.3
			clampedShift := math.Max(-maxShift, math.Min(maxShift, rawShift))
			finalHue = math.Mod(targetHue+clampedShift+1, 1)
		}
		sat = math.Max(0.25, math.Min(0.85, sat))
		light = math.Max(0.25, math.Min(0.75, light))
		r, g, b := hslToRGB(finalHue, sat, light)
		return rgbToHex(r, g, b)
	}

	return map[string]string{
		"term0":  bg,
		"term1":  harmonize(0.98, normalSat, normalLight),
		"term2":  harmonize(0.36, normalSat, normalLight),
		"term3":  harmonize(0.12, normalSat+0.10, normalLight),
		"term4":  harmonize(0.58, normalSat, normalLight),
		"term5":  harmonize(0.85, normalSat, normalLight),
		"term6":  harmonize(0.48, normalSat, normalLight),
		"term7":  c.OnSurfaceVariant,
		"term8":  c.Outline,
		"term9":  harmonize(0.98, brightSat, brightLight),
		"term10": harmonize(0.36, brightSat, brightLight),
		"term11": harmonize(0.12, brightSat+0.10, brightLight),
		"term12": harmonize(0.58, brightSat, brightLight),
		"term13": harmonize(0.85, brightSat, brightLight),
		"term14": harmonize(0.48, brightSat, brightLight),
		"term15": c.OnBackground,
	}
}

func hexToRGB(hex string) (float64, float64, float64) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	var r, g, b uint8
	fmt.Sscanf(hex[:2], "%02x", &r)
	fmt.Sscanf(hex[2:4], "%02x", &g)
	fmt.Sscanf(hex[4:6], "%02x", &b)
	return float64(r) / 255, float64(g) / 255, float64(b) / 255
}

func rgbToHex(r, g, b float64) string {
	ri := uint8(math.Round(r * 255))
	gi := uint8(math.Round(g * 255))
	bi := uint8(math.Round(b * 255))
	return fmt.Sprintf("#%02X%02X%02X", ri, gi, bi)
}

func rgbToHSL(r, g, b float64) (float64, float64, float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l := (max + min) / 2

	if max == min {
		return 0, 0, l
	}

	var h, s float64
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	switch {
	case max == r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case max == g:
		h = (b-r)/d + 2
	case max == b:
		h = (r-g)/d + 4
	}
	h /= 6

	return h, s, l
}

func hslToRGB(h, s, l float64) (float64, float64, float64) {
	if s == 0 {
		return l, l, l
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	hueToRGB := func(p, q, t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		if t < 1.0/6.0 {
			return p + (q-p)*6*t
		}
		if t < 1.0/2.0 {
			return q
		}
		if t < 2.0/3.0 {
			return p + (q-p)*(2.0/3.0-t)*6
		}
		return p
	}

	r := hueToRGB(p, q, h+1.0/3.0)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-1.0/3.0)

	return r, g, b
}

func writeSchemeJSON(path string, data interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func writeSchemeSCSS(path string, c *presets.PresetColors, colors map[string]string, termColors map[string]string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("$darkmode: %v;\n", c.Darkmode))
	sb.WriteString(fmt.Sprintf("$transparent: %v;\n", c.Transparent))

	for k, v := range colors {
		sb.WriteString(fmt.Sprintf("$%s: %s;\n", k, v))
	}
	for k, v := range termColors {
		sb.WriteString(fmt.Sprintf("$%s: %s;\n", k, v))
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func modeString(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

func termSource(explicit bool) string {
	if explicit {
		return "preset-explicit"
	}
	return "harmonized"
}
