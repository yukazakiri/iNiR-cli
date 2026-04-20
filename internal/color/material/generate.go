package material

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"reflect"
	"strings"
)

type GenerateOptions struct {
	ImagePath          string
	SeedColor          string
	Mode               string
	Scheme             string
	Harmony            float64
	TermSaturation     float64
	TermBrightness     float64
	TermBgBrightness   float64
	HarmonizeThreshold float64
	TermFgBoost        float64
	ColorStrength      float64
	Soften             bool
	BlendBgFg          bool
}

type GenerateResult struct {
	SeedColor      string
	Mode           string
	Scheme         string
	MaterialColors map[string]string
	TermColors     map[string]string
	DarkPalette    map[string]string
	LightPalette   map[string]string
	Palette        map[string]string
	Meta           map[string]interface{}
	SourcePath     string
}

func GeneratePalette(opts GenerateOptions) (*GenerateResult, error) {
	var sourceARGB int
	var sourceHCT HCT
	var sourcePath string

	if opts.SeedColor != "" {
		sourceARGB = HexToARGB(opts.SeedColor)
		sourceHCT = HCTFromARGB(sourceARGB)
	} else if opts.ImagePath != "" {
		img, err := loadImage(opts.ImagePath)
		if err != nil {
			return nil, fmt.Errorf("load image: %w", err)
		}
		img = ScaleImage(img, 128)
		sourceARGB = ExtractSeedColor(img)
		sourceHCT = HCTFromARGB(sourceARGB)
		sourcePath = opts.ImagePath
	} else {
		return nil, fmt.Errorf("no image or color provided")
	}

	schemeType := opts.Scheme
	if strings.HasPrefix(schemeType, "scheme-") {
		schemeType = strings.TrimPrefix(schemeType, "scheme-")
	}
	if schemeType == "vibrant" && sourceHCT.Chroma < 20 {
		schemeType = "neutral"
	}

	isDark := opts.Mode == "dark"
	scheme := GenerateScheme(sourceHCT, isDark, "scheme-"+schemeType)

	materialColors := extractColorsFromScheme(scheme, opts)

	if scheme.IsDark {
		materialColors["success"] = "#B5CCBA"
		materialColors["onSuccess"] = "#213528"
		materialColors["successContainer"] = "#374B3E"
		materialColors["onSuccessContainer"] = "#D1E9D6"
	} else {
		materialColors["success"] = "#4F6354"
		materialColors["onSuccess"] = "#FFFFFF"
		materialColors["successContainer"] = "#D1E8D5"
		materialColors["onSuccessContainer"] = "#0C1F13"
	}

	termColors := generateTerminalColors(materialColors, sourceARGB, opts)

	darkScheme := GenerateScheme(sourceHCT, true, "scheme-"+schemeType)
	lightScheme := GenerateScheme(sourceHCT, false, "scheme-"+schemeType)
	darkPalette := extractColorsFromScheme(darkScheme, opts)
	lightPalette := extractColorsFromScheme(lightScheme, opts)

	if darkScheme.IsDark {
		darkPalette["success"] = "#B5CCBA"
		darkPalette["onSuccess"] = "#213528"
		darkPalette["successContainer"] = "#374B3E"
		darkPalette["onSuccessContainer"] = "#D1E9D6"
	}
	if !lightScheme.IsDark {
		lightPalette["success"] = "#4F6354"
		lightPalette["onSuccess"] = "#FFFFFF"
		lightPalette["successContainer"] = "#D1E8D5"
		lightPalette["onSuccessContainer"] = "#0C1F13"
	}

	palette := buildPaletteJSON(materialColors)

	meta := map[string]interface{}{
		"source":              sourcePath,
		"seed_color":          ARGBToHex(sourceARGB),
		"mode":                opts.Mode,
		"scheme":              "scheme-" + schemeType,
		"term_harmony":        opts.Harmony,
		"term_saturation":     opts.TermSaturation,
		"term_brightness":     opts.TermBrightness,
		"term_bg_brightness":  opts.TermBgBrightness,
		"term_fg_boost":       opts.TermFgBoost,
		"harmonize_threshold": opts.HarmonizeThreshold,
		"color_strength":      opts.ColorStrength,
		"blend_bg_fg":         opts.BlendBgFg,
		"soften":              opts.Soften,
		"generated_by":        "inir-cli",
	}

	result := &GenerateResult{
		SeedColor:      ARGBToHex(sourceARGB),
		Mode:           opts.Mode,
		Scheme:         "scheme-" + schemeType,
		MaterialColors: materialColors,
		TermColors:     termColors,
		DarkPalette:    darkPalette,
		LightPalette:   lightPalette,
		Palette:        palette,
		Meta:           meta,
		SourcePath:     sourcePath,
	}

	return result, nil
}

func extractColorsFromScheme(scheme *Scheme, opts GenerateOptions) map[string]string {
	colors := make(map[string]string)
	v := reflect.ValueOf(scheme).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		if field.Type() == reflect.TypeOf(HCT{}) {
			hct := field.Interface().(HCT)
			hex := hct.ToHex()

			if opts.Soften && opts.Scheme != "scheme-tonal-spot" && opts.Scheme != "scheme-neutral" && opts.Scheme != "scheme-monochrome" {
				hct = HCTFromHCT(hct.Hue, hct.Chroma*0.60, hct.Tone)
				hex = hct.ToHex()
			}

			if opts.ColorStrength != 1.0 && hct.Chroma > 2.0 {
				hct = HCTFromHCT(hct.Hue, hct.Chroma*opts.ColorStrength, hct.Tone)
				hex = hct.ToHex()
			}

			colors[fieldName] = hex

			snake := camelToSnake(fieldName)
			if snake != fieldName {
				colors[snake] = hex
			}
		}
	}

	colors["sourceColor"] = ARGBToHex(0)
	return colors
}

func camelToSnake(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

func buildPaletteJSON(colors map[string]string) map[string]string {
	keys := []string{
		"primary", "on_primary", "primary_container", "on_primary_container",
		"primary_fixed", "primary_fixed_dim", "on_primary_fixed", "on_primary_fixed_variant",
		"secondary", "on_secondary", "secondary_container", "on_secondary_container",
		"secondary_fixed", "secondary_fixed_dim", "on_secondary_fixed", "on_secondary_fixed_variant",
		"tertiary", "on_tertiary", "tertiary_container", "on_tertiary_container",
		"tertiary_fixed", "tertiary_fixed_dim", "on_tertiary_fixed", "on_tertiary_fixed_variant",
		"error", "on_error", "error_container", "on_error_container",
		"background", "on_background", "surface", "on_surface",
		"surface_dim", "surface_bright", "surface_variant", "on_surface_variant",
		"surface_container_lowest", "surface_container_low", "surface_container",
		"surface_container_high", "surface_container_highest",
		"outline", "outline_variant",
		"inverse_surface", "inverse_on_surface", "inverse_primary",
		"shadow", "scrim", "surface_tint",
		"success", "on_success", "success_container", "on_success_container",
	}

	palette := make(map[string]string)
	for _, key := range keys {
		if val, ok := colors[key]; ok {
			palette[key] = val
		}
	}
	return palette
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (r *GenerateResult) WriteJSON(colorsPath, palettePath, terminalPath, metaPath string) error {
	colorsJSON := make(map[string]string)
	for k, v := range r.Palette {
		colorsJSON[k] = v
	}
	for k, v := range r.TermColors {
		colorsJSON[k] = v
	}

	if err := writeJSONFile(colorsPath, colorsJSON); err != nil {
		return err
	}
	if err := writeJSONFile(palettePath, r.Palette); err != nil {
		return err
	}
	if err := writeJSONFile(terminalPath, r.TermColors); err != nil {
		return err
	}
	if err := writeJSONFile(metaPath, r.Meta); err != nil {
		return err
	}
	return nil
}

func (r *GenerateResult) WriteTerminalJSON(path string) error {
	return writeJSONFile(path, r.TermColors)
}

func (r *GenerateResult) WriteSCSS(path string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("$darkmode: %v;\n", r.Mode == "dark"))
	sb.WriteString("$transparent: false;\n")
	for k, v := range r.MaterialColors {
		sb.WriteString(fmt.Sprintf("$%s: %s;\n", k, v))
	}
	for k, v := range r.TermColors {
		sb.WriteString(fmt.Sprintf("$%s: %s;\n", k, v))
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeJSONFile(path string, data interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
