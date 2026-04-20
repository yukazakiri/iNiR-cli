package cmd

import (
	"github.com/snowarch/inir-cli/internal/target"
	"github.com/snowarch/inir-cli/internal/target/gtk"
	"github.com/snowarch/inir-cli/internal/target/terminal"
	"github.com/snowarch/inir-cli/internal/target/chrome"
	"github.com/snowarch/inir-cli/internal/target/editor"
	"github.com/snowarch/inir-cli/internal/target/spicetify"
	"github.com/snowarch/inir-cli/internal/target/steam"
	"github.com/snowarch/inir-cli/internal/target/vesktop"
	"github.com/snowarch/inir-cli/internal/target/pear"
	"github.com/snowarch/inir-cli/internal/target/sddm"
)

func init() {
	target.Register("gtk-kde", func() target.Applier { return &gtk.Applier{} })
	target.Register("terminals", func() target.Applier { return &terminal.Applier{} })
	target.Register("chrome", func() target.Applier { return &chrome.Applier{} })
	target.Register("editors", func() target.Applier { return &editor.Applier{} })
	target.Register("zed", func() target.Applier { return &editor.ZedApplier{} })
	target.Register("spicetify", func() target.Applier { return &spicetify.Applier{} })
	target.Register("steam", func() target.Applier { return &steam.Applier{} })
	target.Register("vesktop", func() target.Applier { return &vesktop.Applier{} })
	target.Register("pear-desktop", func() target.Applier { return &pear.Applier{} })
	target.Register("sddm", func() target.Applier { return &sddm.Applier{} })
}
