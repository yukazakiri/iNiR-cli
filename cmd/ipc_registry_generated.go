// File: ipc_registry_generated.go
//
// AUTO-GENERATED from upstream iNiR IPC registry snapshots.
// Do not manually edit — regenerate from upstream instead.
// Manual customizations should live in ipc_registry_overrides.go.
//
// Contains:
//   - generatedIPCKebabAliases: kebab-case → camelCase alias map
//   - generatedIPCTargets:      full target definitions with functions
package cmd

var generatedIPCKebabAliases = map[string]string{
	"alt-switcher":         "altSwitcher",
	"app-catalog":          "appCatalog",
	"cliphist-service":     "cliphistService",
	"close-confirm":        "closeConfirm",
	"control-panel":        "controlPanel",
	"coverflow-selector":   "coverflowSelector",
	"global-actions":       "globalActions",
	"media-controls":       "mediaControls",
	"osd-volume":           "osdVolume",
	"package-search":       "packageSearch",
	"panel-family":         "panelFamily",
	"shell-update":         "shellUpdate",
	"sidebar-left":         "sidebarLeft",
	"sidebar-right":        "sidebarRight",
	"voice-search":         "voiceSearch",
	"waction-center":       "wactionCenter",
	"waffle-alt-switcher":  "waffleAltSwitcher",
	"wallpaper-selector":   "wallpaperSelector",
	"wnotification-center": "wnotificationCenter",
}

var generatedIPCTargets = []ipcTarget{
	target("ai", "shared", "AI chat service. Multi-provider (Gemini, OpenAI, Mistral) with tool support.", funcs(
		fn("ensureInitialized", "", "Force-load models and API keys"),
		fn("diagnose", "", "Dump current AI state as JSON"),
		fn("run", "<inputText>", "Send a message or command to the AI chat"),
		fn("runGet", "<inputText>", "Run AI command and return the last response"),
	)),
	target("altSwitcher", "shared", "Alt+Tab window switcher.", panelFunctions("Open switcher", "Close switcher", "Toggle switcher", "Focus next window", "Focus previous window"), `bind "Alt+Tab" { spawn "inir" "altSwitcher" "next"; }`),
	target("appCatalog", "shared", "App catalog service. Browse, search, and install curated applications.", funcs(
		fn("refresh", "", "Refresh installed-state cache"),
		fn("search", "<query>", "Filter catalog entries by query"),
		fn("install", "<id>", "Install app by catalog ID"),
		fn("list", "", "List catalog apps"),
	)),
	target("audio", "shared", "Volume and mute control.", funcs(
		fn("volumeUp", "", "Increase volume"),
		fn("volumeDown", "", "Decrease volume"),
		fn("mute", "", "Toggle speaker mute"),
		fn("micMute", "", "Toggle microphone mute"),
	)),
	target("bar", "shared", "Top bar visibility.", toggleCloseOpen("Show/hide bar", "Hide bar", "Show bar")),
	target("brightness", "shared", "Display brightness control.", funcs(
		fn("increment", "", "Increase brightness"),
		fn("decrement", "", "Decrease brightness"),
	)),
	target("cheatsheet", "shared", "Keyboard shortcuts reference.", toggleCloseOpen("Open/close cheatsheet", "Hide cheatsheet overlay", "Show cheatsheet overlay"), `bind "Super+Slash" { spawn "inir" "cheatsheet" "toggle"; }`),
	target("clipboard", "shared", "Clipboard history panel.", funcs(
		fn("open", "", "Open panel"),
		fn("close", "", "Close panel"),
		fn("toggle", "", "Open/close panel"),
	), `bind "Super+V" { spawn "inir" "clipboard" "toggle"; }`),
	target("cliphistService", "shared", "Clipboard history service.", funcs(fn("update", "", "Refresh clipboard history"))),
	target("closeConfirm", "shared", "Close window confirmation dialog.", funcs(
		fn("trigger", "", "Show close confirmation for focused window"),
		fn("close", "", "Dismiss the dialog without closing"),
	)),
	target("controlPanel", "shared", "Quick settings panel.", toggleCloseOpen("Open/close control panel", "Close control panel", "Open control panel")),
	target("coverflowSelector", "shared", "Wallpaper coverflow picker.", toggleOpenClose("Open/close coverflow selector", "Open coverflow selector", "Close coverflow selector")),
	target("gamemode", "shared", "Performance mode for gaming.", funcs(
		fn("toggle", "", "Toggle gamemode on/off"),
		fn("activate", "", "Force enable gamemode"),
		fn("deactivate", "", "Force disable gamemode"),
		fn("status", "", "Print current gamemode state"),
	), `bind "Super+F12" { spawn "inir" "gamemode" "toggle"; }`),
	target("globalActions", "shared", "Command palette / action registry.", funcs(
		fn("run", "<actionId> <args>", "Execute action by ID"),
		fn("list", "<category>", "List all actions"),
		fn("search", "<query>", "Fuzzy search actions"),
		fn("open", "", "Open overview in action mode"),
	)),
	target("keyboard", "shared", "Keyboard layout switching.", funcs(
		fn("switchLayout", "", "Switch to next keyboard layout"),
		fn("switchLayoutPrevious", "", "Switch to previous keyboard layout"),
		fn("getCurrentLayout", "", "Get current layout name"),
		fn("getLayouts", "", "Get all configured layout names"),
	)),
	target("lock", "shared", "Lock screen.", funcs(
		fn("activate", "", "Lock the screen"),
		fn("deactivate", "", "Cancel lock"),
		fn("status", "", "Return lock state"),
		fn("focus", "", "Refocus lock screen input"),
	)),
	target("mediaControls", "shared", "Floating media controls panel.", toggleCloseOpen("Open/close media controls", "Hide media controls", "Show media controls")),
	target("minimize", "shared", "Window minimization.", funcs(
		fn("minimize", "", "Minimize focused window"),
		fn("restore", "<windowId>", "Restore a minimized window by ID"),
	)),
	target("mpris", "shared", "Media player control.", funcs(
		fn("pauseAll", "", "Pause all players"),
		fn("playPause", "", "Toggle play/pause"),
		fn("previous", "", "Previous track"),
		fn("next", "", "Next track"),
	)),
	target("notifications", "shared", "Notification management.", funcs(
		fn("test", "", "Send test notifications"),
		fn("clearAll", "", "Dismiss all notifications"),
		fn("toggleSilent", "", "Toggle Do Not Disturb mode"),
	)),
	target("osd", "waffle", "Waffle on-screen display indicator.", funcs(fn("trigger", "", "Show OSD indicator"))),
	target("osdVolume", "shared", "On-screen volume indicator.", funcs(
		fn("trigger", "", "Show volume OSD"),
		fn("hide", "", "Hide volume OSD"),
		fn("toggle", "", "Toggle volume OSD"),
	)),
	target("osk", "shared", "On-screen keyboard.", toggleCloseOpen("Show/hide on-screen keyboard", "Hide on-screen keyboard", "Show on-screen keyboard")),
	target("overlay", "ii", "The central overlay.", funcs(fn("toggle", "", "Open/close overlay")), `bind "Super+G" { spawn "inir" "overlay" "toggle"; }`),
	target("overview", "shared", "Toggle the workspace overview panel.", funcs(
		fn("toggle", "", "Open/close overview"),
		fn("close", "", "Close overview"),
		fn("open", "", "Open overview"),
		fn("toggleReleaseInterrupt", "", "Clear super-key release interrupt"),
		fn("clipboardToggle", "", "Open clipboard search, or close if already open"),
		fn("actionOpen", "", "Open overview in action search mode"),
	), `bind "Mod+Space" { spawn "inir" "overview" "toggle"; }`),
	target("packageSearch", "shared", "Package search service.", funcs(
		fn("search", "<query>", "Start a package search"),
		fn("results", "", "Print current search results"),
	)),
	target("panelFamily", "shared", "Switch between panel styles.", funcs(
		fn("cycle", "", "Cycle to next panel family"),
		fn("set", "<family>", "Set specific family"),
	)),
	target("region", "shared", "Region selection tools.", funcs(
		fn("screenshot", "", "Take a region screenshot"),
		fn("search", "", "Image search"),
		fn("googleLens", "", "Start region capture for Google Lens"),
		fn("ocr", "", "OCR text recognition"),
		fn("record", "", "Record region without audio"),
		fn("recordWithSound", "", "Record region with audio"),
	)),
	target("search", "waffle", "Waffle start menu / search.", toggleCloseOpen("Open/close start menu", "Close start menu", "Open start menu")),
	target("session", "shared", "Power menu.", toggleCloseOpen("Open/close session menu", "Hide session screen", "Show session screen")),
	target("settings", "shared", "Open the settings window.", funcs(
		fn("open", "", "Open settings window"),
		fn("toggle", "", "Toggle settings"),
	)),
	target("shellUpdate", "shared", "Shell update checker.", funcs(
		fn("toggle", "", "Open/close update overlay"),
		fn("open", "", "Open update overlay"),
		fn("close", "", "Close update overlay"),
		fn("check", "", "Check for updates now"),
		fn("performUpdate", "", "Run the update"),
		fn("dismiss", "", "Dismiss update notification"),
		fn("undismiss", "", "Un-dismiss update notification"),
		fn("diagnose", "", "Dump update state as JSON"),
	)),
	target("sidebarLeft", "shared", "Left sidebar.", toggleCloseOpen("Open/close left sidebar", "Hide left sidebar", "Show left sidebar")),
	target("sidebarRight", "shared", "Right sidebar.", toggleCloseOpen("Open/close right sidebar", "Hide right sidebar", "Show right sidebar")),
	target("taskview", "waffle", "Waffle task view.", toggleCloseOpen("Open/close task view", "Hide task view", "Open task view")),
	target("tiling", "shared", "Tiling layout overlay.", funcs(
		fn("toggle", "", "Open/close tiling picker"),
		fn("open", "", "Open tiling picker"),
		fn("hide", "", "Close picker and OSD"),
		fn("cycle", "", "Cycle to next tiling preset"),
		fn("showOsd", "", "Flash current tiling preset OSD"),
		fn("promote", "", "Promote focused window to master position"),
	)),
	target("voiceSearch", "shared", "Voice search using Gemini API.", funcs(
		fn("start", "", "Start recording"),
		fn("stop", "", "Stop recording"),
		fn("toggle", "", "Toggle recording"),
	)),
	target("wactionCenter", "waffle", "Waffle action center.", funcs(fn("toggle", "", "Open/close action center"))),
	target("waffleAltSwitcher", "waffle", "Waffle Alt+Tab window switcher.", panelFunctions("Open switcher", "Close switcher", "Toggle switcher", "Focus next window", "Focus previous window")),
	target("wallpaperSelector", "shared", "Wallpaper picker grid.", funcs(
		fn("toggle", "", "Open/close wallpaper selector"),
		fn("open", "", "Open wallpaper selector"),
		fn("close", "", "Close wallpaper selector"),
		fn("toggleOnMonitor", "<monitorName>", "Open selector on a specific monitor"),
		fn("random", "", "Pick a random wallpaper"),
	)),
	target("wbar", "waffle", "Waffle taskbar visibility.", toggleCloseOpen("Show/hide taskbar", "Hide taskbar", "Show taskbar")),
	target("wnotificationCenter", "waffle", "Waffle notification center.", funcs(fn("toggle", "", "Open/close notification center"))),
	target("wwidgets", "waffle", "Waffle widgets panel.", toggleCloseOpen("Open/close widgets", "Close widgets", "Open widgets")),
	target("ytmusic", "shared", "Direct YtMusic player control.", funcs(
		fn("playPause", "", "Toggle YtMusic play/pause"),
		fn("next", "", "Play next track"),
		fn("previous", "", "Play previous track"),
		fn("stop", "", "Stop playback"),
	)),
	target("zoom", "shared", "Screen zoom.", funcs(
		fn("zoomIn", "", "Increase zoom level"),
		fn("zoomOut", "", "Decrease zoom level"),
	)),
}

func target(name, family, description string, functions []ipcFunction, example ...string) ipcTarget {
	meta := ipcTarget{Name: name, Family: family, Description: description, Functions: functions}
	if len(example) > 0 {
		meta.Example = example[0]
	}
	return meta
}

func funcs(functions ...ipcFunction) []ipcFunction {
	return functions
}

func fn(name, args, description string) ipcFunction {
	return ipcFunction{Name: name, Args: args, Description: description}
}

func toggleCloseOpen(toggleDescription, closeDescription, openDescription string) []ipcFunction {
	return funcs(
		fn("toggle", "", toggleDescription),
		fn("close", "", closeDescription),
		fn("open", "", openDescription),
	)
}

func toggleOpenClose(toggleDescription, openDescription, closeDescription string) []ipcFunction {
	return funcs(
		fn("toggle", "", toggleDescription),
		fn("open", "", openDescription),
		fn("close", "", closeDescription),
	)
}

func panelFunctions(openDescription, closeDescription, toggleDescription, nextDescription, previousDescription string) []ipcFunction {
	return funcs(
		fn("open", "", openDescription),
		fn("close", "", closeDescription),
		fn("toggle", "", toggleDescription),
		fn("next", "", nextDescription),
		fn("previous", "", previousDescription),
	)
}
