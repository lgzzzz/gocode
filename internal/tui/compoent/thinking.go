package compoent

import (
	"strings"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui/markdown"
)

type ThinkingMessage struct {
	id          string
	content     string
	renderCache string
	renderWidth int
	dirty       bool
}

func NewThinkingMessage(id, content string) *ThinkingMessage {
	m := &ThinkingMessage{id: id}
	m.SetContent(content)
	return m
}

func (m *ThinkingMessage) Type() string    { return string(agent.MsgThinking) }
func (m *ThinkingMessage) MsgID() string   { return m.id }
func (m *ThinkingMessage) Content() string { return m.content }

func (m *ThinkingMessage) SetContent(content string) {
	if m.content == content {
		return
	}
	m.content = content
	m.dirty = true
}

func (m *ThinkingMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = m.renderMarkdown(width)
	m.dirty = false
	return m.renderCache
}

// renderMarkdown converts the thinking content from markdown to styled
// terminal output using glamour (quiet mode, no colors), then prepends
// the thinking bar prefix to each line.
func (m *ThinkingMessage) renderMarkdown(width int) string {
	content := strings.TrimSpace(m.content)
	if content == "" {
		return ""
	}

	renderer := markdown.MarkdownRenderer(width)
	defer renderer.Close()
	out, err := renderer.Render(content)
	if err != nil {
		return renderTrim(thinkingStyle, width, content)
	}

	return renderTrim(thinkingStyle, width, out)
}
