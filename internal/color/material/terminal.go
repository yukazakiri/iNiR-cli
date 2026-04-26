package material

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

var defaultDarkTerms = map[string]string{
	"term0": "#1a1b1e", "term1": "#FF6B6B", "term2": "#6ECF6E", "term3": "#E8C55A",
	"term4": "#6AACDF", "term5": "#C78DD9", "term6": "#5DC8C2", "term7": "#b0a99f",
	"term8": "#5c5f66", "term9": "#FF9E9E", "term10": "#96E496", "term11": "#F5D98A",
	"term12": "#9ECCF5", "term13": "#DEB0F0", "term14": "#8BE8E3", "term15": "#d4cdc3",
}

var defaultLightTerms = map[string]string{
	"term0": "#faf8f5", "term1": "#c75050", "term2": "#5a8a5a", "term3": "#b07840",
	"term4": "#4a7aa0", "term5": "#9060a0", "term6": "#4a9a97", "term7": "#4a4540",
	"term8": "#8a857d", "term9": "#d76060", "term10": "#6a9a6a", "term11": "#c08850",
	"term12": "#5a8ab0", "term13": "#a070b0", "term14": "#5aaaa7", "term15": "#3a3530",
}

func generateTerminalColors(materialColors map[string]string, primaryARGB int, opts GenerateOptions) map[string]string {
	isDark := opts.Mode == "dark"

	termSource := defaultDarkTerms
	if !isDark {
		termSource = defaultLightTerms
	}

	if opts.TermSchemePath != "" {
		if loaded := loadTermScheme(opts.TermSchemePath, isDark); loaded != nil {
			termSource = loaded
		}
	}

	if opts.Scheme == "scheme-monochrome" {
		result := make(map[string]string)
		for k, v := range termSource {
			result[k] = v
		}
		return result
	}

	primaryKeyColor := getPrimaryColor(materialColors)
	primaryColorARGB := HexToARGB(primaryKeyColor)

	result := make(map[string]string)

	surfaceLevels := []struct {
		name  string
		level float64
	}{
		{"background", 0.0}, {"surfaceContainerLowest", 0.2},
		{"surfaceContainerLow", 0.4}, {"surfaceContainer", 0.6},
		{"surfaceContainerHigh", 0.8}, {"surfaceContainerHighest", 1.0},
	}

	getSurface := func(brightness float64) string {
		for i, sl := range surfaceLevels {
			if brightness <= sl.level || i == len(surfaceLevels)-1 {
				if i == 0 {
					return materialColors[sl.name]
				}
				prev := surfaceLevels[i-1]
				var t float64
				if sl.level != prev.level {
					t = (brightness - prev.level) / (sl.level - prev.level)
				}
				c1 := HexToARGB(materialColors[prev.name])
				c2 := HexToARGB(materialColors[sl.name])
				r1, g1, b1 := (c1>>16)&0xFF, (c1>>8)&0xFF, c1&0xFF
				r2, g2, b2 := (c2>>16)&0xFF, (c2>>8)&0xFF, c2&0xFF
				r := int(float64(r1) + float64(r2-r1)*t)
				g := int(float64(g1) + float64(g2-g1)*t)
				b := int(float64(b1) + float64(b2-b1)*t)
				return fmt.Sprintf("#%02X%02X%02X", r, g, b)
			}
		}
		return materialColors["surfaceContainerLow"]
	}

	for colorName, val := range termSource {
		switch colorName {
		case "term0":
			result[colorName] = getSurface(opts.TermBgBrightness)
			continue
		case "term15":
			result[colorName] = getVal(materialColors, "onSurface", "#e0e0e0")
			continue
		case "term8":
			if isDark {
				result[colorName] = getVal(materialColors, "outline", getSurface(minFloat(1.0, opts.TermBgBrightness+0.45)))
			} else {
				result[colorName] = getVal(materialColors, "outlineVariant", getSurface(maxFloat(0.0, opts.TermBgBrightness-0.45)))
			}
			continue
		case "term7":
			harmonized := harmonize(HexToARGB(val), primaryColorARGB, opts.HarmonizeThreshold*0.3, opts.Harmony*0.4)
			harmonized = boostChromaTone(harmonized, opts.TermSaturation*1.2, 1.0)
			result[colorName] = ARGBToHex(harmonized)
			continue
		}

		harmonized := harmonize(HexToARGB(val), primaryColorARGB, opts.HarmonizeThreshold*0.12, opts.Harmony)

		toneMult := 1.0 + ((opts.TermBrightness - 0.5) * 0.8)
		if isDark {
			fgBoost := opts.TermFgBoost * 0.25
			toneMult += fgBoost
		} else {
			fgBoost := opts.TermFgBoost * 0.25 * -1
			toneMult += fgBoost
		}
		if toneMult < 0.60 {
			toneMult = 0.60
		}
		if toneMult > 1.45 {
			toneMult = 1.45
		}

		harmonized = boostChromaTone(harmonized, opts.TermSaturation*2.0, toneMult)
		harmonized = ensureMinChroma(harmonized, 40)

		if opts.Soften && opts.Scheme != "scheme-tonal-spot" && opts.Scheme != "scheme-neutral" && opts.Scheme != "scheme-monochrome" {
			harmonized = boostChromaTone(harmonized, 0.55, 1.0)
		}

		result[colorName] = ARGBToHex(harmonized)
	}

	if bgHex, ok := result["term0"]; ok {
		bgARGB := HexToARGB(bgHex)
		normalColors := []string{"term1", "term2", "term3", "term4", "term5", "term6"}
		for _, c := range normalColors {
			if fgHex, ok := result[c]; ok {
				adjusted := ensureContrast(HexToARGB(fgHex), bgARGB, 4.5, isDark)
				result[c] = ARGBToHex(adjusted)
			}
		}
		brightColors := []string{"term9", "term10", "term11", "term12", "term13", "term14"}
		for _, c := range brightColors {
			if fgHex, ok := result[c]; ok {
				adjusted := ensureContrast(HexToARGB(fgHex), bgARGB, 3.5, isDark)
				result[c] = ARGBToHex(adjusted)
			}
		}
	}

	return result
}

func loadTermScheme(path string, isDark bool) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var root map[string]map[string]string
	if err := json.Unmarshal(data, &root); err != nil {
		// Try top-level map directly (legacy format)
		var direct map[string]string
		if err := json.Unmarshal(data, &direct); err != nil {
			return nil
		}
		return direct
	}
	modeKey := "light"
	if isDark {
		modeKey = "dark"
	}
	if terms, ok := root[modeKey]; ok {
		return terms
	}
	return nil
}

func getPrimaryColor(colors map[string]string) string {
	for _, key := range []string{"primaryPaletteKeyColor", "primary_palette_key_color", "primary"} {
		if v, ok := colors[key]; ok && v != "" {
			return v
		}
	}
	return "#6750A4"
}

func getVal(m map[string]string, key, fallback string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return fallback
}

func harmonize(designColor, sourceColor int, threshold, harmony float64) int {
	fromHCT := HCTFromARGB(designColor)
	toHCT := HCTFromARGB(sourceColor)

	diffDeg := DifferenceDegrees(fromHCT.Hue, toHCT.Hue)
	rotDeg := diffDeg * harmony
	if rotDeg > threshold {
		rotDeg = threshold
	}
	dir := RotationDirection(fromHCT.Hue, toHCT.Hue)
	outputHue := SanitizeDegreesDouble(fromHCT.Hue + rotDeg*dir)

	return HCTFromHCT(outputHue, fromHCT.Chroma, fromHCT.Tone).ToARGB()
}

func boostChromaTone(argb int, chroma, tone float64) int {
	hct := HCTFromARGB(argb)
	newTone := hct.Tone * tone
	if newTone > 95.0 {
		newTone = 95.0
	}
	return HCTFromHCT(hct.Hue, hct.Chroma*chroma, newTone).ToARGB()
}

func ensureMinChroma(argb int, minChroma float64) int {
	hct := HCTFromARGB(argb)
	if hct.Chroma < minChroma {
		return HCTFromHCT(hct.Hue, minChroma, hct.Tone).ToARGB()
	}
	return argb
}

func ensureContrast(fgARGB, bgARGB int, minRatio float64, isDark bool) int {
	currentRatio := contrastRatio(fgARGB, bgARGB)
	if currentRatio >= minRatio {
		return fgARGB
	}

	hct := HCTFromARGB(fgARGB)
	toneLimit := 88.0
	if !isDark {
		toneLimit = 20.0
	}

	bestARGB := fgARGB
	bestRatio := currentRatio

	step := 0.25
	tone := hct.Tone
	for iter := 0; iter < 400; iter++ {
		candidate := HCTFromHCT(hct.Hue, hct.Chroma, tone).ToARGB()
		ratio := contrastRatio(candidate, bgARGB)

		if ratio > bestRatio {
			bestARGB = candidate
			bestRatio = ratio
		}

		if ratio >= minRatio {
			return candidate
		}

		if isDark {
			tone += step
			if tone > toneLimit {
				break
			}
		} else {
			tone -= step
			if tone < toneLimit {
				break
			}
		}
	}

	return bestARGB
}

func contrastRatio(fgARGB, bgARGB int) float64 {
	l1 := relativeLuminance(fgARGB)
	l2 := relativeLuminance(bgARGB)
	lighter := math.Max(l1, l2)
	darker := math.Min(l1, l2)
	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(argb int) float64 {
	r := float64((argb>>16)&0xFF) / 255.0
	g := float64((argb>>8)&0xFF) / 255.0
	b := float64(argb&0xFF) / 255.0

	linearize := func(c float64) float64 {
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}

	return 0.2126*linearize(r) + 0.7152*linearize(g) + 0.0722*linearize(b)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
