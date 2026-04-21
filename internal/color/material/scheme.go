package material

type Scheme struct {
	IsDark bool
	SourceColorHCT HCT

	Primary            HCT
	OnPrimary          HCT
	PrimaryContainer   HCT
	OnPrimaryContainer HCT
	PrimaryFixed       HCT
	PrimaryFixedDim    HCT
	OnPrimaryFixed     HCT
	OnPrimaryFixedVariant HCT

	Secondary            HCT
	OnSecondary          HCT
	SecondaryContainer   HCT
	OnSecondaryContainer HCT
	SecondaryFixed       HCT
	SecondaryFixedDim    HCT
	OnSecondaryFixed     HCT
	OnSecondaryFixedVariant HCT

	Tertiary            HCT
	OnTertiary          HCT
	TertiaryContainer   HCT
	OnTertiaryContainer HCT
	TertiaryFixed       HCT
	TertiaryFixedDim    HCT
	OnTertiaryFixed     HCT
	OnTertiaryFixedVariant HCT

	Error            HCT
	OnError          HCT
	ErrorContainer   HCT
	OnErrorContainer HCT

	Background   HCT
	OnBackground HCT
	Surface      HCT
	OnSurface    HCT

	SurfaceVariant      HCT
	OnSurfaceVariant    HCT
	Outline             HCT
	OutlineVariant      HCT

	SurfaceDim              HCT
	SurfaceBright           HCT
	SurfaceContainerLowest  HCT
	SurfaceContainerLow     HCT
	SurfaceContainer        HCT
	SurfaceContainerHigh    HCT
	SurfaceContainerHighest HCT

	InverseSurface    HCT
	InverseOnSurface  HCT
	InversePrimary    HCT

	Shadow      HCT
	Scrim       HCT
	SurfaceTint HCT
}

type TonalPalette struct {
	Hue    float64
	Chroma float64
}

func NewTonalPalette(hue, chroma float64) TonalPalette {
	return TonalPalette{Hue: hue, Chroma: chroma}
}

func (tp TonalPalette) Tone(tone float64) HCT {
	return HCTFromHCT(tp.Hue, tp.Chroma, tone)
}

type CorePalettes struct {
	Primary   TonalPalette
	Secondary TonalPalette
	Tertiary  TonalPalette
	Neutral   TonalPalette
	NeutralVariant TonalPalette
	Error     TonalPalette
}

func NewCorePalettes(sourceHCT HCT) CorePalettes {
	h := sourceHCT.Hue
	c := sourceHCT.Chroma

	return CorePalettes{
		Primary:       NewTonalPalette(h, mathMax(c, 48.0)),
		Secondary:     NewTonalPalette(h, c*0.3+8.0),
		Tertiary:      NewTonalPalette(SanitizeDegreesDouble(h+60.0), c*0.4+16.0),
		Neutral:       NewTonalPalette(h, c*0.1+4.0),
		NeutralVariant: NewTonalPalette(h, c*0.15+8.0),
		Error:         NewTonalPalette(25.0, 84.0),
	}
}

func GenerateScheme(sourceHCT HCT, isDark bool, schemeType string) *Scheme {
	cp := NewCorePalettes(sourceHCT)

	switch schemeType {
	case "scheme-tonal-spot":
		return tonalSpotScheme(cp, sourceHCT, isDark)
	case "scheme-neutral":
		return neutralScheme(cp, sourceHCT, isDark)
	case "scheme-monochrome":
		return monochromeScheme(cp, sourceHCT, isDark)
	case "scheme-vibrant":
		return vibrantScheme(cp, sourceHCT, isDark)
	case "scheme-expressive":
		return expressiveScheme(cp, sourceHCT, isDark)
	case "scheme-fidelity":
		return fidelityScheme(cp, sourceHCT, isDark)
	case "scheme-content":
		return contentScheme(cp, sourceHCT, isDark)
	case "scheme-rainbow":
		return rainbowScheme(cp, sourceHCT, isDark)
	case "scheme-fruit-salad":
		return fruitSaladScheme(cp, sourceHCT, isDark)
	default:
		return tonalSpotScheme(cp, sourceHCT, isDark)
	}
}

func tonalSpotScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	s := &Scheme{IsDark: dark, SourceColorHCT: src}

	if dark {
		s.Primary = cp.Primary.Tone(80)
		s.OnPrimary = cp.Primary.Tone(20)
		s.PrimaryContainer = cp.Primary.Tone(30)
		s.OnPrimaryContainer = cp.Primary.Tone(90)
		s.PrimaryFixed = cp.Primary.Tone(90)
		s.PrimaryFixedDim = cp.Primary.Tone(80)
		s.OnPrimaryFixed = cp.Primary.Tone(10)
		s.OnPrimaryFixedVariant = cp.Primary.Tone(30)

		s.Secondary = cp.Secondary.Tone(80)
		s.OnSecondary = cp.Secondary.Tone(20)
		s.SecondaryContainer = cp.Secondary.Tone(30)
		s.OnSecondaryContainer = cp.Secondary.Tone(90)
		s.SecondaryFixed = cp.Secondary.Tone(90)
		s.SecondaryFixedDim = cp.Secondary.Tone(80)
		s.OnSecondaryFixed = cp.Secondary.Tone(10)
		s.OnSecondaryFixedVariant = cp.Secondary.Tone(30)

		s.Tertiary = cp.Tertiary.Tone(80)
		s.OnTertiary = cp.Tertiary.Tone(20)
		s.TertiaryContainer = cp.Tertiary.Tone(30)
		s.OnTertiaryContainer = cp.Tertiary.Tone(90)
		s.TertiaryFixed = cp.Tertiary.Tone(90)
		s.TertiaryFixedDim = cp.Tertiary.Tone(80)
		s.OnTertiaryFixed = cp.Tertiary.Tone(10)
		s.OnTertiaryFixedVariant = cp.Tertiary.Tone(30)

		s.Error = cp.Error.Tone(80)
		s.OnError = cp.Error.Tone(20)
		s.ErrorContainer = cp.Error.Tone(30)
		s.OnErrorContainer = cp.Error.Tone(90)

		s.Background = cp.Neutral.Tone(6)
		s.OnBackground = cp.Neutral.Tone(90)
		s.Surface = cp.Neutral.Tone(6)
		s.OnSurface = cp.Neutral.Tone(90)
		s.SurfaceVariant = cp.NeutralVariant.Tone(30)
		s.OnSurfaceVariant = cp.NeutralVariant.Tone(80)
		s.Outline = cp.NeutralVariant.Tone(60)
		s.OutlineVariant = cp.NeutralVariant.Tone(38)
		s.SurfaceDim = cp.Neutral.Tone(6)
		s.SurfaceBright = cp.Neutral.Tone(24)
		s.SurfaceContainerLowest = cp.Neutral.Tone(4)
		s.SurfaceContainerLow = cp.Neutral.Tone(10)
		s.SurfaceContainer = cp.Neutral.Tone(12)
		s.SurfaceContainerHigh = cp.Neutral.Tone(17)
		s.SurfaceContainerHighest = cp.Neutral.Tone(22)

		s.InverseSurface = cp.Neutral.Tone(90)
		s.InverseOnSurface = cp.Neutral.Tone(20)
		s.InversePrimary = cp.Primary.Tone(40)
		s.Shadow = cp.Neutral.Tone(0)
		s.Scrim = cp.Neutral.Tone(0)
		s.SurfaceTint = cp.Primary.Tone(80)
	} else {
		s.Primary = cp.Primary.Tone(40)
		s.OnPrimary = cp.Primary.Tone(100)
		s.PrimaryContainer = cp.Primary.Tone(90)
		s.OnPrimaryContainer = cp.Primary.Tone(10)
		s.PrimaryFixed = cp.Primary.Tone(90)
		s.PrimaryFixedDim = cp.Primary.Tone(80)
		s.OnPrimaryFixed = cp.Primary.Tone(10)
		s.OnPrimaryFixedVariant = cp.Primary.Tone(30)

		s.Secondary = cp.Secondary.Tone(40)
		s.OnSecondary = cp.Secondary.Tone(100)
		s.SecondaryContainer = cp.Secondary.Tone(90)
		s.OnSecondaryContainer = cp.Secondary.Tone(10)
		s.SecondaryFixed = cp.Secondary.Tone(90)
		s.SecondaryFixedDim = cp.Secondary.Tone(80)
		s.OnSecondaryFixed = cp.Secondary.Tone(10)
		s.OnSecondaryFixedVariant = cp.Secondary.Tone(30)

		s.Tertiary = cp.Tertiary.Tone(40)
		s.OnTertiary = cp.Tertiary.Tone(100)
		s.TertiaryContainer = cp.Tertiary.Tone(90)
		s.OnTertiaryContainer = cp.Tertiary.Tone(10)
		s.TertiaryFixed = cp.Tertiary.Tone(90)
		s.TertiaryFixedDim = cp.Tertiary.Tone(80)
		s.OnTertiaryFixed = cp.Tertiary.Tone(10)
		s.OnTertiaryFixedVariant = cp.Tertiary.Tone(30)

		s.Error = cp.Error.Tone(40)
		s.OnError = cp.Error.Tone(100)
		s.ErrorContainer = cp.Error.Tone(90)
		s.OnErrorContainer = cp.Error.Tone(10)

		s.Background = cp.Neutral.Tone(99)
		s.OnBackground = cp.Neutral.Tone(10)
		s.Surface = cp.Neutral.Tone(99)
		s.OnSurface = cp.Neutral.Tone(10)
		s.SurfaceVariant = cp.NeutralVariant.Tone(90)
		s.OnSurfaceVariant = cp.NeutralVariant.Tone(30)
		s.Outline = cp.NeutralVariant.Tone(50)
		s.OutlineVariant = cp.NeutralVariant.Tone(80)
		s.SurfaceDim = cp.Neutral.Tone(87)
		s.SurfaceBright = cp.Neutral.Tone(98)
		s.SurfaceContainerLowest = cp.Neutral.Tone(100)
		s.SurfaceContainerLow = cp.Neutral.Tone(96)
		s.SurfaceContainer = cp.Neutral.Tone(94)
		s.SurfaceContainerHigh = cp.Neutral.Tone(92)
		s.SurfaceContainerHighest = cp.Neutral.Tone(90)

		s.InverseSurface = cp.Neutral.Tone(20)
		s.InverseOnSurface = cp.Neutral.Tone(95)
		s.InversePrimary = cp.Primary.Tone(80)
		s.Shadow = cp.Neutral.Tone(0)
		s.Scrim = cp.Neutral.Tone(0)
		s.SurfaceTint = cp.Primary.Tone(40)
	}

	return s
}

func neutralScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	cp.Primary = NewTonalPalette(src.Hue, 12.0)
	cp.Secondary = NewTonalPalette(src.Hue, 8.0)
	cp.Tertiary = NewTonalPalette(src.Hue, 16.0)
	return tonalSpotScheme(cp, src, dark)
}

func monochromeScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	cp.Primary = NewTonalPalette(src.Hue, 0.0)
	cp.Secondary = NewTonalPalette(src.Hue, 0.0)
	cp.Tertiary = NewTonalPalette(src.Hue, 0.0)
	return tonalSpotScheme(cp, src, dark)
}

func vibrantScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	h := src.Hue
	cp.Primary = NewTonalPalette(h, 200.0)
	cp.Secondary = NewTonalPalette(h, 24.0)
	cp.Tertiary = NewTonalPalette(SanitizeDegreesDouble(h+120.0), 96.0)
	return tonalSpotScheme(cp, src, dark)
}

func expressiveScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	h := src.Hue
	cp.Primary = NewTonalPalette(SanitizeDegreesDouble(h+240.0), 55.0)
	cp.Secondary = NewTonalPalette(SanitizeDegreesDouble(h+30.0), 30.0)
	cp.Tertiary = NewTonalPalette(SanitizeDegreesDouble(h+120.0), 50.0)
	return tonalSpotScheme(cp, src, dark)
}

func fidelityScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	cp.Primary = NewTonalPalette(src.Hue, mathMax(src.Chroma, 48.0))
	cp.Secondary = NewTonalPalette(src.Hue, mathMax(src.Chroma*0.5, 24.0))
	cp.Tertiary = NewTonalPalette(SanitizeDegreesDouble(src.Hue+60.0), mathMax(src.Chroma*0.7, 32.0))
	return tonalSpotScheme(cp, src, dark)
}

func contentScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	return fidelityScheme(cp, src, dark)
}

func rainbowScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	h := src.Hue
	cp.Primary = NewTonalPalette(h, 48.0)
	cp.Secondary = NewTonalPalette(SanitizeDegreesDouble(h+80.0), 24.0)
	cp.Tertiary = NewTonalPalette(SanitizeDegreesDouble(h+200.0), 32.0)
	return tonalSpotScheme(cp, src, dark)
}

func fruitSaladScheme(cp CorePalettes, src HCT, dark bool) *Scheme {
	h := src.Hue
	cp.Primary = NewTonalPalette(SanitizeDegreesDouble(h-50.0), 48.0)
	cp.Secondary = NewTonalPalette(SanitizeDegreesDouble(h-50.0), 24.0)
	cp.Tertiary = NewTonalPalette(h, 48.0)
	return tonalSpotScheme(cp, src, dark)
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
