package colorutil

import (
	"fmt"
	"strconv"
	"strings"
)

func NormalizeHexLower(value string) (string, bool) {
	return normalizeHex(value, false)
}

func NormalizeHexUpper(value string) (string, bool) {
	return normalizeHex(value, true)
}

func normalizeHex(value string, upper bool) (string, bool) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "#")
	if len(trimmed) != 6 {
		return "", false
	}
	if _, err := strconv.ParseUint(trimmed, 16, 32); err != nil {
		return "", false
	}
	if upper {
		return "#" + strings.ToUpper(trimmed), true
	}
	return "#" + strings.ToLower(trimmed), true
}

func HexToRGB(value string) (uint8, uint8, uint8, bool) {
	normalized, ok := NormalizeHexLower(value)
	if !ok {
		return 0, 0, 0, false
	}
	hex := strings.TrimPrefix(normalized, "#")
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return uint8(r), uint8(g), uint8(b), true
}

func HexToRGBCSV(value string, spaced bool) (string, bool) {
	r, g, b, ok := HexToRGB(value)
	if !ok {
		return "", false
	}
	if spaced {
		return fmt.Sprintf("%d, %d, %d", r, g, b), true
	}
	return fmt.Sprintf("%d,%d,%d", r, g, b), true
}
