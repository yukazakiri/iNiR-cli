// File: register.go
//
// Built-in target registration. Each target name maps to a factory function
// that returns a fresh Applier instance. Add new targets here after creating
// the implementation in internal/target/<name>/.
//
// Current targets: gtk-kde, terminals, chrome, editors, zed, spicetify,
// steam, vesktop, pear-desktop, sddm.
package cmd

import (
	targetpkg "github.com/yukazakiri/inir-cli/internal/target"
	"github.com/yukazakiri/inir-cli/internal/target/chrome"
	"github.com/yukazakiri/inir-cli/internal/target/editor"
	"github.com/yukazakiri/inir-cli/internal/target/gtk"
	"github.com/yukazakiri/inir-cli/internal/target/pear"
	"github.com/yukazakiri/inir-cli/internal/target/sddm"
	"github.com/yukazakiri/inir-cli/internal/target/spicetify"
	"github.com/yukazakiri/inir-cli/internal/target/steam"
	"github.com/yukazakiri/inir-cli/internal/target/terminal"
	"github.com/yukazakiri/inir-cli/internal/target/vesktop"
)

func init() {
	targetpkg.Register("gtk-kde", func() targetpkg.Applier { return &gtk.Applier{} })
	targetpkg.Register("terminals", func() targetpkg.Applier { return &terminal.Applier{} })
	targetpkg.Register("chrome", func() targetpkg.Applier { return &chrome.Applier{} })
	targetpkg.Register("editors", func() targetpkg.Applier { return &editor.Applier{} })
	targetpkg.Register("zed", func() targetpkg.Applier { return &editor.ZedApplier{} })
	targetpkg.Register("spicetify", func() targetpkg.Applier { return &spicetify.Applier{} })
	targetpkg.Register("steam", func() targetpkg.Applier { return &steam.Applier{} })
	targetpkg.Register("vesktop", func() targetpkg.Applier { return &vesktop.Applier{} })
	targetpkg.Register("pear-desktop", func() targetpkg.Applier { return &pear.Applier{} })
	targetpkg.Register("sddm", func() targetpkg.Applier { return &sddm.Applier{} })
}
