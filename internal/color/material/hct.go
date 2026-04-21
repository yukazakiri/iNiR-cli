package material

import (
	"fmt"
	"math"
)

type HCT struct {
	Hue    float64
	Chroma float64
	Tone   float64
}

func HCTFromARGB(argb int) HCT {
	r := float64((argb>>16)&0xFF) / 255.0
	g := float64((argb>>8)&0xFF) / 255.0
	b := float64(argb&0xFF) / 255.0

	l, a, bv := labFromRGB(r, g, b)
	h, c := hclFromLab(l, a, bv)

	return HCT{Hue: h, Chroma: c, Tone: l * 100.0}
}

func HCTFromInt(argb int) HCT {
	return HCTFromARGB(argb)
}

func HCTFromHCT(h, c, t float64) HCT {
	return HCT{Hue: h, Chroma: c, Tone: t}
}

func (hct HCT) ToARGB() int {
	l := hct.Tone / 100.0
	a, b := labComponentsFromHCL(hct.Hue, hct.Chroma/100.0, l)
	r, g, bv := rgbFromLab(l, a, b)
	return argbFromRGB(clampByte(r*255.0), clampByte(g*255.0), clampByte(bv*255.0))
}

func (hct HCT) ToInt() int {
	return hct.ToARGB()
}

func (hct HCT) ToRGBA() [4]uint8 {
	c := hct.ToARGB()
	return [4]uint8{uint8((c >> 16) & 0xFF), uint8((c >> 8) & 0xFF), uint8(c & 0xFF), 255}
}

func (hct HCT) ToHex() string {
	c := hct.ToARGB()
	r, g, b := uint8((c>>16)&0xFF), uint8((c>>8)&0xFF), uint8(c&0xFF)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func ARGBFromRGB(r, g, b int) int {
	return (255 << 24) | (r << 16) | (g << 8) | b
}

func argbFromRGB(r, g, b int) int {
	return (255 << 24) | (r << 16) | (g << 8) | b
}

func HexToARGB(hex string) int {
	for len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) < 6 {
		return 0
	}
	var r, g, b int
	fmt.Sscanf(hex[0:2], "%02x", &r)
	fmt.Sscanf(hex[2:4], "%02x", &g)
	fmt.Sscanf(hex[4:6], "%02x", &b)
	return ARGBFromRGB(r, g, b)
}

func ARGBToHex(argb int) string {
	r := (argb >> 16) & 0xFF
	g := (argb >> 8) & 0xFF
	b := argb & 0xFF
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func RGBAFromARGB(argb int) [4]uint8 {
	return [4]uint8{
		uint8((argb >> 16) & 0xFF),
		uint8((argb >> 8) & 0xFF),
		uint8(argb & 0xFF),
		uint8((argb >> 24) & 0xFF),
	}
}

func clampByte(f float64) int {
	v := int(math.Round(f))
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func SanitizeDegreesDouble(deg float64) float64 {
	deg = math.Mod(deg, 360.0)
	if deg < 0 {
		deg += 360.0
	}
	return deg
}

func DifferenceDegrees(a, b float64) float64 {
	return 180.0 - math.Abs(math.Abs(a-b)-180.0)
}

func RotationDirection(from, to float64) float64 {
	diff := to - from
	if diff > 180 {
		return -1
	}
	if diff < -180 {
		return 1
	}
	if diff > 0 {
		return 1
	}
	return -1
}
