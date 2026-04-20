package steam

import (
	"fmt"
	"os"

	"github.com/yukazakiri/inir-cli/internal/target"
)

type Applier struct{}

func (a *Applier) Apply(ctx *target.Context) error {
	if !ctx.Config.WallpaperTheming.EnableAdwSteam {
		return nil
	}
	fmt.Fprintf(os.Stderr, "[inir-cli] Steam theming not yet implemented in Go (use shell module)\n")
	return nil
}
