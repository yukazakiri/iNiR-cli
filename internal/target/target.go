package target

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/yukazakiri/inir-cli/internal/config"
)

type Context struct {
	Config       *config.Config
	ColorsPath   string
	PalettePath  string
	TerminalPath string
	SCSSPath     string
	MetaPath     string
	OutputDir    string
}

type Applier interface {
	Apply(ctx *Context) error
}

func (ctx *Context) ReadColorsJSON() (map[string]string, error) {
	return readStringMapJSON(ctx.ColorsPath)
}

func (ctx *Context) ReadPaletteJSON() (map[string]string, error) {
	return readStringMapJSON(ctx.PalettePath)
}

func (ctx *Context) ReadTerminalJSON() (map[string]string, error) {
	return readStringMapJSON(ctx.TerminalPath)
}

func (ctx *Context) ReadMetaJSON() (map[string]interface{}, error) {
	data, err := os.ReadFile(ctx.MetaPath)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ctx *Context) XDGConfigHome() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return dir
}

func (ctx *Context) XDGStateHome() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local/state")
	}
	return dir
}

func (ctx *Context) XDGDataHome() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local/share")
	}
	return dir
}

func (ctx *Context) Home() string {
	home, _ := os.UserHomeDir()
	return home
}

func readStringMapJSON(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for k, v := range raw {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result, nil
}

var registry map[string]func() Applier

func Register(name string, factory func() Applier) {
	if registry == nil {
		registry = make(map[string]func() Applier)
	}
	registry[name] = factory
}

func GetApplier(name string) Applier {
	if f, ok := registry[name]; ok {
		return f()
	}
	return nil
}

func ListTargets() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}
