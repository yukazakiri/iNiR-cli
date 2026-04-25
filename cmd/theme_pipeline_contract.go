package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

type outputContract struct {
	OutputDir    string
	ColorsPath   string
	PalettePath  string
	TerminalPath string
	MetaPath     string
	SCSSPath     string
}

func newOutputContract(outputDir string) outputContract {
	return outputContract{
		OutputDir:    outputDir,
		ColorsPath:   filepath.Join(outputDir, "colors.json"),
		PalettePath:  filepath.Join(outputDir, "palette.json"),
		TerminalPath: filepath.Join(outputDir, "terminal.json"),
		MetaPath:     filepath.Join(outputDir, "theme-meta.json"),
		SCSSPath:     filepath.Join(outputDir, "material_colors.scss"),
	}
}

func (contract outputContract) EnsureDir() error {
	if err := os.MkdirAll(contract.OutputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	return nil
}

func (contract outputContract) RequireColors() error {
	if _, err := os.Stat(contract.ColorsPath); os.IsNotExist(err) {
		return fmt.Errorf("no colors.json found — run 'generate' first")
	}
	return nil
}

func (contract outputContract) WritePresetResult(colorsJSON map[string]string, palette map[string]string, terminal map[string]string, meta map[string]interface{}) error {
	if err := writeSchemeJSON(contract.ColorsPath, colorsJSON); err != nil {
		return fmt.Errorf("write colors.json: %w", err)
	}
	if err := writeSchemeJSON(contract.PalettePath, palette); err != nil {
		return fmt.Errorf("write palette.json: %w", err)
	}
	if err := writeSchemeJSON(contract.TerminalPath, terminal); err != nil {
		return fmt.Errorf("write terminal.json: %w", err)
	}
	if err := writeSchemeJSON(contract.MetaPath, meta); err != nil {
		return fmt.Errorf("write theme-meta.json: %w", err)
	}
	if err := writeChromiumThemeContracts(contract.OutputDir, palette); err != nil {
		return fmt.Errorf("write compatibility files: %w", err)
	}
	return nil
}
