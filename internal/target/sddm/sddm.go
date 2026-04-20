package sddm

import (
	"fmt"
	"os"

	"github.com/yukazakiri/inir-cli/internal/target"
)

type Applier struct{}

func (a *Applier) Apply(ctx *target.Context) error {
	if _, err := os.Stat("/usr/share/sddm/themes/ii-pixel"); os.IsNotExist(err) {
		return nil
	}
	fmt.Fprintf(os.Stderr, "[inir-cli] SDDM sync not yet implemented in Go (use sync-pixel-sddm.py)\n")
	return nil
}
