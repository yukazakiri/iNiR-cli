package material

import "math"

func labFromRGB(r, g, b float64) (float64, float64, float64) {
	linearR := linearize(r)
	linearG := linearize(g)
	linearB := linearize(b)

	x := 0.4124564*linearR + 0.3575761*linearG + 0.1804375*linearB
	y := 0.2126729*linearR + 0.7151522*linearG + 0.0721750*linearB
	z := 0.0193339*linearR + 0.1191920*linearG + 0.9503041*linearB

	xl := x / 0.95047
	yl := y / 1.00000
	zl := z / 1.08883

	fx := labF(xl)
	fy := labF(yl)
	fz := labF(zl)

	l := 116.0*fy - 16.0
	a := 500.0 * (fx - fy)
	bv := 200.0 * (fy - fz)

	return l / 100.0, a, bv
}

func rgbFromLab(l, a, b float64) (float64, float64, float64) {
	l2 := (l*100.0 + 16.0) / 116.0
	fy := l2
	fx := a/500.0 + fy
	fz := fy - b/200.0

	x := 0.95047 * labFInv(fx)
	y := 1.00000 * labFInv(fy)
	z := 1.08883 * labFInv(fz)

	linearR := 3.2404542*x - 1.5371385*y - 0.4985314*z
	linearG := -0.9692660*x + 1.8760108*y + 0.0415560*z
	linearB := 0.0556434*x - 0.2040259*y + 1.0572252*z

	return delinearize(linearR), delinearize(linearG), delinearize(linearB)
}

func hclFromLab(l, a, b float64) (float64, float64) {
	c := math.Sqrt(a*a + b*b)
	var h float64
	if c > 0.0001 {
		h = math.Atan2(b, a) * 180.0 / math.Pi
		if h < 0 {
			h += 360.0
		}
	} else {
		h = 0
	}
	return h, c / 100.0
}

func labComponentsFromHCL(h, c, l float64) (float64, float64) {
	c2 := c * 100.0
	hRad := h * math.Pi / 180.0
	a := c2 * math.Cos(hRad)
	b := c2 * math.Sin(hRad)
	return a, b
}

func linearize(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func delinearize(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

func labF(t float64) float64 {
	if t > 0.008856 {
		return math.Cbrt(t)
	}
	return (903.3*t + 16.0) / 116.0
}

func labFInv(t float64) float64 {
	e := 216.0 / 24389.0
	k := 24389.0 / 27.0
	if t*e > 16 {
		return t * t * t
	}
	return (116.0*t - 16.0) / k
}
