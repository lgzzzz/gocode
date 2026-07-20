// Package terminal provides terminal capability detection and color support,
// mirroring pi's approach in packages/tui/src/terminal-image.ts and
// packages/coding-agent/src/modes/interactive/theme/theme.ts.
//
// Key features:
//   - TrueColor (24-bit) vs 256-color detection via COLORTERM + known terminals
//   - Terminal background color query (OSC 11) for auto light/dark theme
//   - Color type that adapts output based on detected capabilities
package terminal

import (
	"os"
	"strings"
	"sync"
)

// ImageProtocol represents a terminal image protocol.
type ImageProtocol string

const (
	ImageNone   ImageProtocol = ""
	ImageKitty  ImageProtocol = "kitty"
	ImageITerm2 ImageProtocol = "iterm2"
)

// Capabilities describes terminal feature support.
type Capabilities struct {
	Images     ImageProtocol
	TrueColor  bool
	Hyperlinks bool
}

var (
	cachedCapabilities *Capabilities
	capsOnce           sync.Once
)

// DetectCapabilities detects terminal capabilities from environment variables.
// This mirrors pi's detectCapabilities() in terminal-image.ts.
func DetectCapabilities() Capabilities {
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	terminalEmulator := strings.ToLower(os.Getenv("TERMINAL_EMULATOR"))
	term := strings.ToLower(os.Getenv("TERM"))
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))
	hasTrueColorHint := colorTerm == "truecolor" || colorTerm == "24bit"

	// tmux: conservative - images unreliable, hyperlinks depend on tmux config
	if os.Getenv("TMUX") != "" || strings.HasPrefix(term, "tmux") {
		return Capabilities{
			Images:     ImageNone,
			TrueColor:  hasTrueColorHint,
			Hyperlinks: probeTmuxHyperlinks(),
		}
	}

	// screen: no hyperlinks
	if strings.HasPrefix(term, "screen") {
		return Capabilities{
			Images:     ImageNone,
			TrueColor:  hasTrueColorHint,
			Hyperlinks: false,
		}
	}

	// Kitty
	if os.Getenv("KITTY_WINDOW_ID") != "" || termProgram == "kitty" {
		return Capabilities{Images: ImageKitty, TrueColor: true, Hyperlinks: true}
	}

	// Ghostty
	if termProgram == "ghostty" || strings.Contains(term, "ghostty") ||
		os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return Capabilities{Images: ImageKitty, TrueColor: true, Hyperlinks: true}
	}

	// WezTerm
	if os.Getenv("WEZTERM_PANE") != "" || termProgram == "wezterm" {
		return Capabilities{Images: ImageKitty, TrueColor: true, Hyperlinks: true}
	}

	// Warp
	if termProgram == "warpterminal" || os.Getenv("WARP_SESSION_ID") != "" ||
		os.Getenv("WARP_TERMINAL_SESSION_UUID") != "" {
		return Capabilities{Images: ImageKitty, TrueColor: true, Hyperlinks: true}
	}

	// iTerm2
	if os.Getenv("ITERM_SESSION_ID") != "" || termProgram == "iterm.app" {
		return Capabilities{Images: ImageITerm2, TrueColor: true, Hyperlinks: true}
	}

	// Windows Terminal
	if os.Getenv("WT_SESSION") != "" {
		return Capabilities{Images: ImageNone, TrueColor: true, Hyperlinks: true}
	}

	// VS Code
	if termProgram == "vscode" {
		return Capabilities{Images: ImageNone, TrueColor: true, Hyperlinks: true}
	}

	// Alacritty
	if termProgram == "alacritty" {
		return Capabilities{Images: ImageNone, TrueColor: true, Hyperlinks: true}
	}

	// JetBrains
	if terminalEmulator == "jetbrains-jediterm" {
		return Capabilities{Images: ImageNone, TrueColor: true, Hyperlinks: false}
	}

	// Unknown terminal: conservative defaults.
	// Use COLORTERM hint for truecolor; hyperlinks off by default.
	return Capabilities{
		Images:     ImageNone,
		TrueColor:  hasTrueColorHint,
		Hyperlinks: false,
	}
}

// GetCapabilities returns cached terminal capabilities.
func GetCapabilities() Capabilities {
	capsOnce.Do(func() {
		caps := DetectCapabilities()
		cachedCapabilities = &caps
	})
	return *cachedCapabilities
}

// ResetCapabilitiesCache clears the cached capabilities (useful for testing).
func ResetCapabilitiesCache() {
	capsOnce = sync.Once{}
	cachedCapabilities = nil
}
