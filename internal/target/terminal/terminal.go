package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/snowarch/inir-cli/internal/target"
)

type Applier struct{}

func (a *Applier) Apply(ctx *target.Context) error {
	if !ctx.Config.WallpaperTheming.EnableTerminal {
		return nil
	}

	termColors, err := ctx.ReadTerminalJSON()
	if err != nil {
		return fmt.Errorf("read terminal.json: %w", err)
	}

	sequences := buildANSISequences(termColors)
	sequencesDir := filepath.Join(ctx.OutputDir, "terminal")
	os.MkdirAll(sequencesDir, 0755)
	os.WriteFile(filepath.Join(sequencesDir, "sequences.txt"), []byte(sequences), 0644)

	injectSequences(sequences)

	return nil
}

func buildANSISequences(colors map[string]string) string {
	var sb strings.Builder
	for i := 0; i <= 15; i++ {
		key := fmt.Sprintf("term%d", i)
		hex := colors[key]
		hex = strings.TrimPrefix(hex, "#")
		sb.WriteString(fmt.Sprintf("\x1b]4;%d;#%s\x1b\\", i, hex))
	}
	if fg, ok := colors["term7"]; ok {
		sb.WriteString(fmt.Sprintf("\x1b]10;#%s\x1b\\", strings.TrimPrefix(fg, "#")))
	}
	if bg, ok := colors["term0"]; ok {
		sb.WriteString(fmt.Sprintf("\x1b]11;#%s\x1b\\", strings.TrimPrefix(bg, "#")))
	}
	if cur, ok := colors["term7"]; ok {
		sb.WriteString(fmt.Sprintf("\x1b]12;#%s\x1b\\", strings.TrimPrefix(cur, "#")))
	}
	return sb.String()
}

func injectSequences(sequences string) {
	entries, err := os.ReadDir("/dev/pts")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ptsPath := filepath.Join("/dev/pts", entry.Name())
		f, err := os.OpenFile(ptsPath, os.O_WRONLY, 0)
		if err != nil {
			continue
		}
		f.WriteString(sequences)
		f.Close()
	}
}
