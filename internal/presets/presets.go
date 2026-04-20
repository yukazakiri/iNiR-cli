package presets

type Preset struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Tags        []string
	Meta        *PresetMeta
	Colors      PresetColors
}

type PresetMeta struct {
	RoundingScale   float64
	FontStyle       string
	BorderWidthScale float64
}

type PresetColors struct {
	Darkmode    bool
	Transparent bool

	Background                string
	OnBackground              string
	Surface                   string
	SurfaceDim                string
	SurfaceBright             string
	SurfaceContainerLowest    string
	SurfaceContainerLow       string
	SurfaceContainer          string
	SurfaceContainerHigh      string
	SurfaceContainerHighest   string
	OnSurface                 string
	SurfaceVariant            string
	OnSurfaceVariant          string
	InverseSurface            string
	InverseOnSurface          string
	Outline                   string
	OutlineVariant            string
	Shadow                    string
	Scrim                     string
	SurfaceTint               string
	Primary                   string
	OnPrimary                 string
	PrimaryContainer          string
	OnPrimaryContainer        string
	InversePrimary            string
	Secondary                 string
	OnSecondary               string
	SecondaryContainer        string
	OnSecondaryContainer      string
	Tertiary                  string
	OnTertiary                string
	TertiaryContainer         string
	OnTertiaryContainer       string
	Error                     string
	OnError                   string
	ErrorContainer            string
	OnErrorContainer          string
	PrimaryFixed              string
	PrimaryFixedDim           string
	OnPrimaryFixed            string
	OnPrimaryFixedVariant     string
	SecondaryFixed            string
	SecondaryFixedDim         string
	OnSecondaryFixed          string
	OnSecondaryFixedVariant   string
	TertiaryFixed             string
	TertiaryFixedDim          string
	OnTertiaryFixed           string
	OnTertiaryFixedVariant    string
	Success                   string
	OnSuccess                 string
	SuccessContainer          string
	OnSuccessContainer        string

	Term0  string
	Term1  string
	Term2  string
	Term3  string
	Term4  string
	Term5  string
	Term6  string
	Term7  string
	Term8  string
	Term9  string
	Term10 string
	Term11 string
	Term12 string
	Term13 string
	Term14 string
	Term15 string
}

func GetPreset(id string) *Preset {
	for i := range Presets {
		if Presets[i].ID == id {
			return &Presets[i]
		}
	}
	return nil
}

func ListIDs() []string {
	ids := make([]string, len(Presets))
	for i, p := range Presets {
		ids[i] = p.ID
	}
	return ids
}

func (c *PresetColors) HasExplicitTerminalColors() bool {
	return c.Term1 != ""
}

func (c *PresetColors) ToMap() map[string]string {
	m := map[string]string{
		"background":                c.Background,
		"on_background":             c.OnBackground,
		"surface":                   c.Surface,
		"surface_dim":               c.SurfaceDim,
		"surface_bright":            c.SurfaceBright,
		"surface_container_lowest":  c.SurfaceContainerLowest,
		"surface_container_low":     c.SurfaceContainerLow,
		"surface_container":         c.SurfaceContainer,
		"surface_container_high":    c.SurfaceContainerHigh,
		"surface_container_highest": c.SurfaceContainerHighest,
		"on_surface":                c.OnSurface,
		"surface_variant":           c.SurfaceVariant,
		"on_surface_variant":        c.OnSurfaceVariant,
		"inverse_surface":           c.InverseSurface,
		"inverse_on_surface":        c.InverseOnSurface,
		"outline":                   c.Outline,
		"outline_variant":           c.OutlineVariant,
		"shadow":                    c.Shadow,
		"scrim":                     c.Scrim,
		"surface_tint":              c.SurfaceTint,
		"primary":                   c.Primary,
		"on_primary":                c.OnPrimary,
		"primary_container":         c.PrimaryContainer,
		"on_primary_container":      c.OnPrimaryContainer,
		"inverse_primary":           c.InversePrimary,
		"secondary":                 c.Secondary,
		"on_secondary":              c.OnSecondary,
		"secondary_container":       c.SecondaryContainer,
		"on_secondary_container":    c.OnSecondaryContainer,
		"tertiary":                  c.Tertiary,
		"on_tertiary":               c.OnTertiary,
		"tertiary_container":        c.TertiaryContainer,
		"on_tertiary_container":     c.OnTertiaryContainer,
		"error":                     c.Error,
		"on_error":                  c.OnError,
		"error_container":           c.ErrorContainer,
		"on_error_container":        c.OnErrorContainer,
		"primary_fixed":             c.PrimaryFixed,
		"primary_fixed_dim":         c.PrimaryFixedDim,
		"on_primary_fixed":          c.OnPrimaryFixed,
		"on_primary_fixed_variant":  c.OnPrimaryFixedVariant,
		"secondary_fixed":           c.SecondaryFixed,
		"secondary_fixed_dim":       c.SecondaryFixedDim,
		"on_secondary_fixed":        c.OnSecondaryFixed,
		"on_secondary_fixed_variant": c.OnSecondaryFixedVariant,
		"tertiary_fixed":            c.TertiaryFixed,
		"tertiary_fixed_dim":        c.TertiaryFixedDim,
		"on_tertiary_fixed":         c.OnTertiaryFixed,
		"on_tertiary_fixed_variant": c.OnTertiaryFixedVariant,
		"success":                   c.Success,
		"on_success":                c.OnSuccess,
		"success_container":         c.SuccessContainer,
		"on_success_container":      c.OnSuccessContainer,
	}
	return m
}

func (c *PresetColors) TerminalColorMap() map[string]string {
	if !c.HasExplicitTerminalColors() {
		return nil
	}
	return map[string]string{
		"term0":  c.Term0,
		"term1":  c.Term1,
		"term2":  c.Term2,
		"term3":  c.Term3,
		"term4":  c.Term4,
		"term5":  c.Term5,
		"term6":  c.Term6,
		"term7":  c.Term7,
		"term8":  c.Term8,
		"term9":  c.Term9,
		"term10": c.Term10,
		"term11": c.Term11,
		"term12": c.Term12,
		"term13": c.Term13,
		"term14": c.Term14,
		"term15": c.Term15,
	}
}
