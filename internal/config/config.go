package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	PanelFamily             string                   `json:"panelFamily"`
	Theme                   string                   `json:"theme"`
	Appearance              Appearance               `json:"appearance"`
	Background              Background               `json:"background"`
	WallpaperTheming        WallpaperTheming         `json:"wallpaperTheming"`
	SoftenColors            bool                     `json:"softenColors"`
	TerminalColorAdjustments TerminalColorAdjustments `json:"terminalColorAdjustments"`
	TerminalGenerationProps  TerminalGenerationProps  `json:"terminalGenerationProps"`
}

type Appearance struct {
	Palette         Palette            `json:"palette"`
	WallpaperTheming WallpaperTheming   `json:"wallpaperTheming"`
	SoftenColors    bool               `json:"softenColors"`
}

type Palette struct {
	Type       string `json:"type"`
	AccentColor string `json:"accentColor"`
}

type WallpaperTheming struct {
	EnableAppsAndShell   bool                       `json:"enableAppsAndShell"`
	EnableTerminal       bool                       `json:"enableTerminal"`
	EnableQtApps         bool                       `json:"enableQtApps"`
	EnableVesktop        bool                       `json:"enableVesktop"`
	EnableZed            bool                       `json:"enableZed"`
	EnableVSCode         bool                       `json:"enableVSCode"`
	EnableChrome         bool                       `json:"enableChrome"`
	EnableSpicetify      bool                       `json:"enableSpicetify"`
	EnableAdwSteam       bool                       `json:"enableAdwSteam"`
	EnablePearDesktop    bool                       `json:"enablePearDesktop"`
	EnableNeovim         bool                       `json:"enableNeovim"`
	EnableOpenCode       bool                       `json:"enableOpenCode"`
	ColorStrength        float64                    `json:"colorStrength"`
	Terminals            map[string]bool            `json:"terminals"`
	VscodeEditors        map[string]bool            `json:"vscodeEditors"`
	UseBackdropForColors bool                       `json:"useBackdropForColors"`
	TerminalGenerationProps TerminalGenerationProps `json:"terminalGenerationProps"`
}

type TerminalColorAdjustments struct {
	Saturation         float64 `json:"saturation"`
	Brightness         float64 `json:"brightness"`
	Harmony            float64 `json:"harmony"`
	BackgroundBrightness float64 `json:"backgroundBrightness"`
}

type TerminalGenerationProps struct {
	ForceDarkMode     bool    `json:"forceDarkMode"`
	HarmonizeThreshold float64 `json:"harmonizeThreshold"`
	TermFgBoost       float64 `json:"termFgBoost"`
	Harmony           float64 `json:"harmony"`
}

type Background struct {
	WallpaperPath      string               `json:"wallpaperPath"`
	ThumbnailPath      string               `json:"thumbnailPath"`
	WallpapersByMonitor []MonitorWallpaper  `json:"wallpapersByMonitor"`
	Backdrop           Backdrop             `json:"backdrop"`
}

type MonitorWallpaper struct {
	Monitor      string `json:"monitor"`
	Path         string `json:"path"`
	WorkspaceFirst int  `json:"workspaceFirst"`
	WorkspaceLast  int  `json:"workspaceLast"`
}

type Backdrop struct {
	UseMainWallpaper string `json:"useMainWallpaper"`
	WallpaperPath    string `json:"wallpaperPath"`
	ThumbnailPath    string `json:"thumbnailPath"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.WallpaperTheming = cfg.Appearance.WallpaperTheming
	cfg.SoftenColors = cfg.Appearance.SoftenColors
	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Theme:     "auto",
		PanelFamily: "ii",
		SoftenColors: false,
		WallpaperTheming: WallpaperTheming{
			EnableAppsAndShell: true,
			EnableTerminal:     true,
			EnableQtApps:       true,
			EnableVesktop:      true,
			EnableZed:          true,
			EnableVSCode:       true,
			EnableChrome:       true,
			EnableSpicetify:    false,
			EnableAdwSteam:     false,
			EnablePearDesktop:  false,
			EnableNeovim:       false,
			EnableOpenCode:     false,
			ColorStrength:      1.0,
		},
		TerminalColorAdjustments: TerminalColorAdjustments{
			Saturation:         0.65,
			Brightness:         0.60,
			Harmony:            0.40,
			BackgroundBrightness: 0.50,
		},
		TerminalGenerationProps: TerminalGenerationProps{
			HarmonizeThreshold: 100,
			TermFgBoost:        0.35,
		},
	}
}
