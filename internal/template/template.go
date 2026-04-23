package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yukazakiri/inir-cli/internal/color/material"
)

type TemplateEntry struct {
	Name         string `json:"name"`
	Input        string `json:"input"`
	Output       string `json:"output"`
	templatePath string
	outputPath   string
}

func RenderAll(templateDir string, result *material.GenerateResult) error {
	manifestPath := filepath.Join(templateDir, "templates.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read templates.json: %w", err)
	}

	var manifest struct {
		Templates []TemplateEntry `json:"templates"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse templates.json: %w", err)
	}

	templatesBase := filepath.Join(templateDir, "templates")

	varRe := regexp.MustCompile(`\{\{\s*(.*?)\s*\}\}`)

	rendered := 0
	for _, entry := range manifest.Templates {
		tplPath := filepath.Join(templatesBase, entry.Input)
		outPath := expandHome(entry.Output)

		if _, err := os.Stat(tplPath); os.IsNotExist(err) {
			continue
		}

		content, err := os.ReadFile(tplPath)
		if err != nil {
			continue
		}

		rendered_content := varRe.ReplaceAllStringFunc(string(content), func(match string) string {
			expr := strings.TrimSpace(match[2 : len(match)-2])
			if expr == "image" {
				return result.SourcePath
			}
			parts := strings.Split(expr, ".")
			if len(parts) == 4 && parts[0] == "colors" {
				token := parts[1]
				mode := parts[2]
				prop := parts[3]

				var colorMap map[string]string
				switch mode {
				case "dark":
					colorMap = result.DarkPalette
				case "light":
					colorMap = result.LightPalette
				case "default":
					colorMap = result.MaterialColors
				default:
					colorMap = result.MaterialColors
				}

				hex, ok := colorMap[token]
				if !ok {
					hex, ok = colorMap[camelToSnake(token)]
				}
				if !ok {
					return match
				}

				switch prop {
				case "hex":
					return hex
				case "hex_stripped":
					return strings.TrimPrefix(hex, "#")
				case "rgb":
					argb := material.HexToARGB(hex)
					r := (argb >> 16) & 0xFF
					g := (argb >> 8) & 0xFF
					b := argb & 0xFF
					return fmt.Sprintf("%d, %d, %d", r, g, b)
				}
			}
			return match
		})

		os.MkdirAll(filepath.Dir(outPath), 0755)
		if os.IsExist(err) {
			if fi, _ := os.Lstat(outPath); fi != nil && fi.Mode()&os.ModeSymlink != 0 {
				os.Remove(outPath)
			}
		}
		os.WriteFile(outPath, []byte(rendered_content), 0644)
		rendered++
	}

	if rendered > 0 {
		fmt.Fprintf(os.Stderr, "[inir-cli] Rendered %d template(s)\n", rendered)
	}
	return nil
}

func buildColorsNamespace(result *material.GenerateResult) map[string]map[string]string {
	return map[string]map[string]string{
		"dark":    result.DarkPalette,
		"light":   result.LightPalette,
		"default": result.MaterialColors,
	}
}

func camelToSnake(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
