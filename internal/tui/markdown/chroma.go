package markdown

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
)

// Formatter returns a custom chroma formatter that uses Lipgloss for
// foreground styling while applying a forced background color. This is
// registered with chroma under the name "gocode" so glamour can use it
// for syntax highlighting code blocks.
func Formatter(bgColor color.Color) chroma.Formatter {
	return chroma.FormatterFunc(func(w io.Writer, style *chroma.Style, it chroma.Iterator) error {
		for token := it(); token != chroma.EOF; token = it() {
			entry := style.Get(token.Type)
			if entry.IsZero() {
				if _, err := fmt.Fprint(w, token.Value); err != nil {
					return err
				}
				continue
			}

			s := lipgloss.NewStyle().Background(bgColor)

			if entry.Bold == chroma.Yes {
				s = s.Bold(true)
			}
			if entry.Underline == chroma.Yes {
				s = s.Underline(true)
			}
			if entry.Italic == chroma.Yes {
				s = s.Italic(true)
			}
			if entry.Colour.IsSet() {
				s = s.Foreground(lipgloss.Color(entry.Colour.String()))
			}

			if _, err := fmt.Fprint(w, s.Render(token.Value)); err != nil {
				return err
			}
		}
		return nil
	})
}
