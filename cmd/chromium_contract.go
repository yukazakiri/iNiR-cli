package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func writeChromiumThemeContracts(outputDir string, colors map[string]string) error {
	seedHex, ok := pickChromiumThemeSeed(colors)
	if !ok {
		return nil
	}

	if err := writeChromiumThemeContract(outputDir, seedHex); err != nil {
		return err
	}

	return writeColorSeedContract(outputDir, seedHex)
}

func pickChromiumThemeSeed(colors map[string]string) (string, bool) {
	for _, key := range []string{"surface_container_low", "surface", "background"} {
		if seed, ok := normalizeContractHex(colors[key]); ok {
			return seed, true
		}
	}
	return "", false
}

func normalizeContractHex(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 7 || trimmed[0] != '#' {
		return "", false
	}

	for _, c := range trimmed[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", false
		}
	}

	return strings.ToUpper(trimmed[:1]) + strings.ToUpper(trimmed[1:]), true
}

func writeChromiumThemeContract(outputDir string, hexColor string) error {
	r, g, b, ok := hexToContractRGB(hexColor)
	if !ok {
		return fmt.Errorf("invalid chromium.theme seed color: %s", hexColor)
	}

	value := fmt.Sprintf("%d,%d,%d\n", r, g, b)
	return os.WriteFile(filepath.Join(outputDir, "chromium.theme"), []byte(value), 0644)
}

func writeColorSeedContract(outputDir string, hexColor string) error {
	seed, ok := normalizeContractHex(hexColor)
	if !ok {
		return fmt.Errorf("invalid color.txt seed color: %s", hexColor)
	}
	return os.WriteFile(filepath.Join(outputDir, "color.txt"), []byte(seed+"\n"), 0644)
}

func hexToContractRGB(hexColor string) (int, int, int, bool) {
	normalized, ok := normalizeContractHex(hexColor)
	if !ok {
		return 0, 0, 0, false
	}

	hexValue := strings.TrimPrefix(normalized, "#")
	r, err := strconv.ParseInt(hexValue[0:2], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	g, err := strconv.ParseInt(hexValue[2:4], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	b, err := strconv.ParseInt(hexValue[4:6], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}

	return int(r), int(g), int(b), true
}
