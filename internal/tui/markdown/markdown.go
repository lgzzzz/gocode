package markdown

import (
	"image/color"
	"sync"

	"charm.land/glamour/v2"
	"github.com/alecthomas/chroma/v2/formatters"
)

const formatterName = "gocode"

func init() {
	// Register our custom chroma formatter so glamour can find it by name.
	// Glamour does not offer an option to pass the formatter directly,
	// so we must register it globally.
	var zero color.Color
	formatters.Register(formatterName, Formatter(zero))
}

// MarkdownStyleConfig defines the color and decoration settings for
// markdown elements rendered by glamour.
type MarkdownStyleConfig struct {
	// Base colors
	Foreground string
	Background string

	// Heading colors (h1 through h6)
	H1Color string
	H2Color string
	H3Color string

	// Inline styles
	BoldColor      string
	ItalicColor    string
	CodeColor      string
	CodeBackground string
	LinkColor      string
	LinkUnderline  bool

	// Block styles
	BlockQuoteColor string
	ListMarkerColor string
	HRColor         string

	// Code block
	CodeBlockBackground string
	CodeBlockBorder     string

	// Table
	TableBorderColor string
}

// DefaultStyles returns a sensible default dark-theme markdown style
// configuration. Colors are chosen to work well on dark terminal
// backgrounds (ANSI 16-255 or hex).
func DefaultStyles() MarkdownStyleConfig {
	return MarkdownStyleConfig{
		Foreground: "#EEEEEE",
		Background: "",

		H1Color: "#FF6B6B",
		H2Color: "#FFA94D",
		H3Color: "#FFD43B",

		BoldColor:      "#FFFFFF",
		ItalicColor:    "#EEEEEE",
		CodeColor:      "#FF6B6B",
		CodeBackground: "#2A2A2A",
		LinkColor:      "#74C0FC",
		LinkUnderline:  true,

		BlockQuoteColor: "#666666",
		ListMarkerColor: "#FFA94D",
		HRColor:         "#444444",

		CodeBlockBackground: "#1E1E1E",
		CodeBlockBorder:     "#444444",

		TableBorderColor: "#444444",
	}
}

// MarkdownRenderer returns a glamour TermRenderer configured with the
// given styles and width. Renderers are memoized per width and shared
// across callers.
//
// The returned renderer is NOT safe for concurrent Render calls
// (goldmark's BlockStack carries state across the public Render API).
// The TUI is single-threaded so this is safe in production.
func MarkdownRenderer(cfg MarkdownStyleConfig, width int) *glamour.TermRenderer {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	if r, ok := mdCache[width]; ok {
		return r
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(buildGlamourStyle(cfg)),
		glamour.WithWordWrap(width),
		glamour.WithChromaFormatter(formatterName),
	)
	mdCache[width] = r
	return r
}

// QuietMarkdownRenderer returns a glamour TermRenderer with no colors
// (plain text with structure) for rendering thinking/reasoning content.
func QuietMarkdownRenderer(cfg MarkdownStyleConfig, width int) *glamour.TermRenderer {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	if r, ok := quietMDCache[width]; ok {
		return r
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(buildQuietGlamourStyle(cfg)),
		glamour.WithWordWrap(width),
		glamour.WithChromaFormatter(formatterName),
	)
	quietMDCache[width] = r
	return r
}

// InvalidateCache drops all cached renderers. Call this when the style
// configuration changes (e.g. theme switching).
func InvalidateCache() {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	mdCache = map[int]*glamour.TermRenderer{}
	quietMDCache = map[int]*glamour.TermRenderer{}
}

var (
	mdCacheMu    sync.Mutex
	mdCache      = map[int]*glamour.TermRenderer{}
	quietMDCache = map[int]*glamour.TermRenderer{}
)
