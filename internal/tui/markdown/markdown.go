package markdown

import (
	"sync"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

var darkCompactConfig = styles.DraculaStyleConfig

func init() {
	darkCompactConfig.Document.Margin = new(uint(0))
	darkCompactConfig.H1.Prefix = ""
	darkCompactConfig.H2.Prefix = ""
	darkCompactConfig.H3.Prefix = ""
	darkCompactConfig.H4.Prefix = ""
	darkCompactConfig.H5.Prefix = ""
	darkCompactConfig.H6.Prefix = ""
}

// MarkdownRenderer returns a glamour TermRenderer configured with compact table
// styles and the given word-wrap width. Renderers are memoized per width.
func MarkdownRenderer(width int) *glamour.TermRenderer {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	if r, ok := mdCache[width]; ok {
		return r
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(darkCompactConfig),
		glamour.WithWordWrap(width),
	)
	mdCache[width] = r
	return r
}

func InvalidateCache() {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	mdCache = map[int]*glamour.TermRenderer{}
}

var (
	mdCacheMu sync.Mutex
	mdCache   = map[int]*glamour.TermRenderer{}
)
