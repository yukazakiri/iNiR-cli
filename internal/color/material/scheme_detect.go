package material

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

func DetectSchemeFromImage(imagePath string) (string, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var totalColorfulness float64
	var satSum float64
	var satCount float64
	var hueSpread [360]int
	var totalHue float64
	var hueCount float64
	pixels := 0

	step := 1
	totalPixels := w * h
	if totalPixels > 100000 {
		step = totalPixels / 100000
		if step < 1 {
			step = 1
		}
	}

	for y := 0; y < h; y += step {
		for x := 0; x < w; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			rf := float64(r>>8) / 255.0
			gf := float64(g>>8) / 255.0
			bf := float64(b>>8) / 255.0

			rg := absF(rf - gf)
			yb := absF((rf+gf)/2.0 - bf)
			totalColorfulness += sqrtF(rg*rg+yb*yb) * 0.5

			h, s, _ := rgbToHSL(rf, gf, bf)
			if s > 0.08 {
				hueSpread[int(h*360)%360]++
				totalHue += h
				hueCount++
			}
			satSum += s
			satCount++
			pixels++
		}
	}

	if pixels == 0 {
		return "scheme-tonal-spot", nil
	}

	avgColorfulness := totalColorfulness / float64(pixels) * 100
	avgSaturation := satSum / float64(satCount) * 255

	hueVar := 0.0
	if hueCount > 0 {
		meanHue := totalHue / hueCount
		for i := 0; i < 360; i++ {
			if hueSpread[i] > 0 {
				diff := float64(i)/360.0 - meanHue
				if diff > 0.5 {
					diff -= 1.0
				}
				if diff < -0.5 {
					diff += 1.0
				}
				hueVar += float64(hueSpread[i]) * diff * diff
			}
		}
		hueVar = sqrtF(hueVar/hueCount) * 57.3
	}

	return pickScheme(avgColorfulness, avgSaturation, hueVar), nil
}

func pickScheme(colorfulness, saturation, hueSpread float64) string {
	if saturation < 20 {
		return "scheme-monochrome"
	}
	if colorfulness < 30 {
		if saturation < 55 {
			return "scheme-neutral"
		}
		if hueSpread < 22 {
			return "scheme-content"
		}
		return "scheme-tonal-spot"
	}
	if colorfulness < 55 {
		if hueSpread < 22 && saturation < 100 {
			return "scheme-content"
		}
		return "scheme-tonal-spot"
	}
	if colorfulness < 90 {
		if saturation > 140 && hueSpread < 35 {
			return "scheme-fidelity"
		}
		if hueSpread < 30 {
			return "scheme-content"
		}
		return "scheme-tonal-spot"
	}
	if hueSpread > 55 && saturation > 150 {
		return "scheme-rainbow"
	}
	if saturation > 160 {
		return "scheme-fidelity"
	}
	if hueSpread > 45 {
		return "scheme-expressive"
	}
	return "scheme-tonal-spot"
}

func rgbToHSL(r, g, b float64) (float64, float64, float64) {
	maxV := maxF(r, g, b)
	minV := minF(r, g, b)
	l := (maxV + minV) / 2.0

	if maxV == minV {
		return 0, 0, l
	}

	d := maxV - minV
	var s float64
	if l > 0.5 {
		s = d / (2.0 - maxV - minV)
	} else {
		s = d / (maxV + minV)
	}

	var h float64
	switch maxV {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h /= 6.0

	return h, s, l
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func sqrtF(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func maxF(a, b, c float64) float64 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

func minF(a, b, c float64) float64 {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
