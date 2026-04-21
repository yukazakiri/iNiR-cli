package material

import (
	"image"
	"math"
	"sort"
)

func QuantizeCelebi(img image.Image, maxColors int) map[int]int {
	bounds := img.Bounds()
	pixels := make([]int, 0, bounds.Dx()*bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8 := int(r >> 8)
			g8 := int(g >> 8)
			b8 := int(b >> 8)
			pixels = append(pixels, ARGBFromRGB(r8, g8, b8))
		}
	}

	return quantize(pixels, maxColors)
}

func quantize(pixels []int, maxColors int) map[int]int {
	if len(pixels) == 0 {
		return map[int]int{}
	}

	const redBits = 5
	const greenBits = 5
	const blueBits = 5

	counts := make(map[uint32]int)
	for _, px := range pixels {
		r := (px >> 16) & 0xFF
		g := (px >> 8) & 0xFF
		b := px & 0xFF
		rq := uint32(r >> (8 - redBits))
		gq := uint32(g >> (8 - greenBits))
		bq := uint32(b >> (8 - blueBits))
		key := (rq << (greenBits + blueBits)) | (gq << blueBits) | bq
		counts[key]++
	}

	type bucket struct {
		key   uint32
		count int
		rSum  int
		gSum  int
		bSum  int
	}

	buckets := make(map[uint32]*bucket)
	for _, px := range pixels {
		r := (px >> 16) & 0xFF
		g := (px >> 8) & 0xFF
		b := px & 0xFF
		rq := uint32(r >> (8 - redBits))
		gq := uint32(g >> (8 - greenBits))
		bq := uint32(b >> (8 - blueBits))
		key := (rq << (greenBits + blueBits)) | (gq << blueBits) | bq

		bt, ok := buckets[key]
		if !ok {
			bt = &bucket{key: key}
			buckets[key] = bt
		}
		bt.count++
		bt.rSum += r
		bt.gSum += g
		bt.bSum += b
	}

	result := make(map[int]int)
	for _, bt := range buckets {
		if bt.count > 0 {
			avgR := bt.rSum / bt.count
			avgG := bt.gSum / bt.count
			avgB := bt.bSum / bt.count
			argb := ARGBFromRGB(avgR, avgG, avgB)
			result[argb] = bt.count
		}
	}

	return result
}

type scoredColor struct {
	argb   int
	score  float64
}

func Score(colors map[int]int, limit int) []int {
	if limit <= 0 {
		limit = 1
	}

	var scored []scoredColor
	for argb, count := range colors {
		hct := HCTFromARGB(argb)
		hue := hct.Hue
		chroma := hct.Chroma
		tone := hct.Tone

		scoreVal := chroma * 100.0
		if chroma < 15 {
			scoreVal = 0
		}
		if tone < 10 || tone > 90 {
			scoreVal *= 0.5
		}

		hueProximity := math.Mod(hue+30, 360.0)
		if hueProximity > 180 {
			hueProximity = 360.0 - hueProximity
		}
		if hueProximity < 15 {
			scoreVal *= (hueProximity / 15.0)
		}

		logCount := math.Log(float64(count) + 1)
		scoreVal *= logCount

		scored = append(scored, scoredColor{argb: argb, score: scoreVal})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	result := make([]int, len(scored))
	for i, s := range scored {
		result[i] = s.argb
	}
	return result
}

func ExtractSeedColor(img image.Image) int {
	colors := QuantizeCelebi(img, 128)
	if len(colors) == 0 {
		return ARGBFromRGB(100, 80, 160)
	}
	scored := Score(colors, 1)
	if len(scored) == 0 {
		return ARGBFromRGB(100, 80, 160)
	}
	return scored[0]
}

func ExtractFromColor(hexColor string) int {
	return HexToARGB(hexColor)
}

func ScaleImage(img image.Image, maxSize int) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	area := w * h
	bitmapArea := maxSize * maxSize

	if area <= bitmapArea {
		return img
	}

	scale := math.Sqrt(float64(bitmapArea) / float64(area))
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	rect := image.Rect(0, 0, newW, newH)
	dst := image.NewRGBA(rect)
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := int(float64(x) / float64(newW) * float64(w))
			srcY := int(float64(y) / float64(newH) * float64(h))
			if srcX >= w {
				srcX = w - 1
			}
			if srcY >= h {
				srcY = h - 1
			}
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst
}
