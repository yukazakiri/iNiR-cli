package color

import "github.com/snowarch/inir-cli/internal/color/material"

type GenerateOptions = material.GenerateOptions
type GenerateResult = material.GenerateResult

func Generate(opts GenerateOptions) (*GenerateResult, error) {
	return material.GeneratePalette(opts)
}

func DetectSchemeFromImage(imagePath string) (string, error) {
	return material.DetectSchemeFromImage(imagePath)
}
