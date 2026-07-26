package markdown

import (
	"sync"

	"charm.land/glamour/v2"
)

func MarkdownRenderer(width int) *glamour.TermRenderer {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	if r, ok := mdCache[width]; ok {
		return r
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
	)
	mdCache[width] = r
	return r
}

// InvalidateCache drops all cached renderers. Call this when the style
// configuration changes (e.g. theme switching).
func InvalidateCache() {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	mdCache = map[int]*glamour.TermRenderer{}
}

var (
	mdCacheMu sync.Mutex
	mdCache   = map[int]*glamour.TermRenderer{}
)
