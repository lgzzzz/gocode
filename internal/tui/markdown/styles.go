package markdown

import (
	"charm.land/glamour/v2/ansi"
)

// ptr returns a pointer to s. Used to satisfy glamour's string-pointer API.
func ptr(s string) *string {
	return &s
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool {
	return &b
}

// uintPtr returns a pointer to u.
func uintPtr(u uint) *uint {
	return &u
}

// buildGlamourStyle converts our MarkdownStyleConfig into a glamour
// ansi.StyleConfig for rendering colored markdown.
func buildGlamourStyle(cfg MarkdownStyleConfig) ansi.StyleConfig {
	bg := ptr(cfg.Background)
	if cfg.Background == "" {
		bg = nil
	}
	codeBg := ptr(cfg.CodeBackground)

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           ptr(cfg.Foreground),
				BackgroundColor: bg,
			},
		},
		BlockQuote: ansi.StyleBlock{
			Indent: uintPtr(1),
			StylePrimitive: ansi.StylePrimitive{
				Color: ptr(cfg.BlockQuoteColor),
			},
		},
		Paragraph: ansi.StyleBlock{},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: ptr(cfg.Foreground),
				},
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: ptr(cfg.H1Color),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: ptr(cfg.H1Color),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: ptr(cfg.H2Color),
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ptr(cfg.H3Color),
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ptr(cfg.H3Color),
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ptr(cfg.H3Color),
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ptr(cfg.H3Color),
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
			Color:  ptr(cfg.ItalicColor),
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolPtr(true),
			Color: ptr(cfg.BoldColor),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color: ptr(cfg.HRColor),
		},
		Item: ansi.StylePrimitive{
			Color: ptr(cfg.ListMarkerColor),
		},
		Link: ansi.StylePrimitive{
			Color:     ptr(cfg.LinkColor),
			Underline: boolPtr(cfg.LinkUnderline),
		},
		LinkText: ansi.StylePrimitive{
			Color:     ptr(cfg.LinkColor),
			Underline: boolPtr(cfg.LinkUnderline),
		},
		Image: ansi.StylePrimitive{
			Color: ptr(cfg.LinkColor),
		},
		ImageText: ansi.StylePrimitive{
			Color: ptr(cfg.LinkColor),
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           ptr(cfg.CodeColor),
				BackgroundColor: ptr(cfg.CodeBackground),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color:           ptr(cfg.Foreground),
					BackgroundColor: codeBg,
				},
			},
			Chroma: &ansi.Chroma{
				Text: ansi.StylePrimitive{
					Color: ptr(cfg.Foreground),
				},
				Error: ansi.StylePrimitive{
					Color: ptr("#FF6B6B"),
				},
				Comment: ansi.StylePrimitive{
					Color: ptr("#6A9955"),
				},
				CommentPreproc: ansi.StylePrimitive{
					Color: ptr("#6A9955"),
				},
				Keyword: ansi.StylePrimitive{
					Color: ptr("#569CD6"),
					Bold:  boolPtr(true),
				},
				KeywordReserved: ansi.StylePrimitive{
					Color: ptr("#569CD6"),
				},
				KeywordNamespace: ansi.StylePrimitive{
					Color: ptr("#569CD6"),
				},
				KeywordType: ansi.StylePrimitive{
					Color: ptr("#4EC9B0"),
				},
				Operator: ansi.StylePrimitive{
					Color: ptr("#D4D4D4"),
				},
				Punctuation: ansi.StylePrimitive{
					Color: ptr("#D4D4D4"),
				},
				Name: ansi.StylePrimitive{
					Color: ptr("#D4D4D4"),
				},
				NameBuiltin: ansi.StylePrimitive{
					Color: ptr("#4EC9B0"),
				},
				NameTag: ansi.StylePrimitive{
					Color: ptr("#569CD6"),
				},
				NameAttribute: ansi.StylePrimitive{
					Color: ptr("#9CDCFE"),
				},
				NameClass: ansi.StylePrimitive{
					Color: ptr("#4EC9B0"),
				},
				NameConstant: ansi.StylePrimitive{
					Color: ptr("#4FC1FF"),
				},
				NameDecorator: ansi.StylePrimitive{
					Color: ptr("#DCDCAA"),
				},
				NameException: ansi.StylePrimitive{
					Color: ptr("#DCDCAA"),
				},
				NameFunction: ansi.StylePrimitive{
					Color: ptr("#DCDCAA"),
				},
				NameOther: ansi.StylePrimitive{
					Color: ptr("#D4D4D4"),
				},
				Literal: ansi.StylePrimitive{
					Color: ptr("#D4D4D4"),
				},
				LiteralNumber: ansi.StylePrimitive{
					Color: ptr("#B5CEA8"),
				},
				LiteralDate: ansi.StylePrimitive{
					Color: ptr("#D4D4D4"),
				},
				LiteralString: ansi.StylePrimitive{
					Color: ptr("#CE9178"),
				},
				LiteralStringEscape: ansi.StylePrimitive{
					Color: ptr("#D7BA7D"),
				},
				GenericDeleted: ansi.StylePrimitive{
					Color: ptr("#FF6B6B"),
				},
				GenericEmph: ansi.StylePrimitive{
					Italic: boolPtr(true),
				},
				GenericInserted: ansi.StylePrimitive{
					Color: ptr("#6A9955"),
				},
				GenericStrong: ansi.StylePrimitive{
					Bold: boolPtr(true),
				},
				GenericSubheading: ansi.StylePrimitive{
					Color: ptr("#569CD6"),
				},
				Background: ansi.StylePrimitive{
					BackgroundColor: codeBg,
				},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: ptr(cfg.Foreground),
				},
			},
		},
	}
}

// buildQuietGlamourStyle converts our MarkdownStyleConfig into a glamour
// ansi.StyleConfig for rendering plain (no-color) markdown. This is used
// for thinking/reasoning content where we want structural formatting
// (headings, lists, code blocks) without color.
func buildQuietGlamourStyle(cfg MarkdownStyleConfig) ansi.StyleConfig {
	fg := ptr(cfg.Foreground)
	codeBg := ptr(cfg.CodeBackground)

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: fg,
			},
		},
		BlockQuote: ansi.StyleBlock{
			Indent: uintPtr(1),
			StylePrimitive: ansi.StylePrimitive{
				Color: fg,
			},
		},
		Paragraph: ansi.StyleBlock{},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: fg,
				},
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: fg,
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: fg,
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: fg,
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: fg,
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: fg,
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: fg,
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: fg,
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
			Color:  fg,
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolPtr(true),
			Color: fg,
		},
		HorizontalRule: ansi.StylePrimitive{
			Color: fg,
		},
		Item: ansi.StylePrimitive{
			Color: fg,
		},
		Link: ansi.StylePrimitive{
			Color:     fg,
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color:     fg,
			Underline: boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color: fg,
		},
		ImageText: ansi.StylePrimitive{
			Color: fg,
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           fg,
				BackgroundColor: codeBg,
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color:           fg,
					BackgroundColor: codeBg,
				},
			},
			Chroma: &ansi.Chroma{
				Text:                ansi.StylePrimitive{Color: fg},
				Error:               ansi.StylePrimitive{Color: fg},
				Comment:             ansi.StylePrimitive{Color: fg},
				CommentPreproc:      ansi.StylePrimitive{Color: fg},
				Keyword:             ansi.StylePrimitive{Color: fg, Bold: boolPtr(true)},
				KeywordReserved:     ansi.StylePrimitive{Color: fg},
				KeywordNamespace:    ansi.StylePrimitive{Color: fg},
				KeywordType:         ansi.StylePrimitive{Color: fg},
				Operator:            ansi.StylePrimitive{Color: fg},
				Punctuation:         ansi.StylePrimitive{Color: fg},
				Name:                ansi.StylePrimitive{Color: fg},
				NameBuiltin:         ansi.StylePrimitive{Color: fg},
				NameTag:             ansi.StylePrimitive{Color: fg},
				NameAttribute:       ansi.StylePrimitive{Color: fg},
				NameClass:           ansi.StylePrimitive{Color: fg},
				NameConstant:        ansi.StylePrimitive{Color: fg},
				NameDecorator:       ansi.StylePrimitive{Color: fg},
				NameException:       ansi.StylePrimitive{Color: fg},
				NameFunction:        ansi.StylePrimitive{Color: fg},
				NameOther:           ansi.StylePrimitive{Color: fg},
				Literal:             ansi.StylePrimitive{Color: fg},
				LiteralNumber:       ansi.StylePrimitive{Color: fg},
				LiteralDate:         ansi.StylePrimitive{Color: fg},
				LiteralString:       ansi.StylePrimitive{Color: fg},
				LiteralStringEscape: ansi.StylePrimitive{Color: fg},
				GenericDeleted:      ansi.StylePrimitive{Color: fg},
				GenericEmph:         ansi.StylePrimitive{Italic: boolPtr(true)},
				GenericInserted:     ansi.StylePrimitive{Color: fg},
				GenericStrong:       ansi.StylePrimitive{Bold: boolPtr(true)},
				GenericSubheading:   ansi.StylePrimitive{Color: fg},
				Background:          ansi.StylePrimitive{BackgroundColor: codeBg},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: fg,
				},
			},
		},
	}
}
