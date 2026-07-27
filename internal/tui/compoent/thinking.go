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
	md          *markdown.Renderer

	markdownCache       []string
	markdownRenderCache []string
}

func NewThinkingMessage(id, content string) *ThinkingMessage {
	m := &ThinkingMessage{id: id, md: markdown.NewRenderer()}
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

func (m *ThinkingMessage) renderMarkdown(width int) string {
	if m.content == "" {
		return ""
	}
	out := m.md.Render(m.content, width-2)
	if m.markdownCache == nil || m.markdownRenderCache == nil {
		outRender := Render(ThinkingStyle, width, out)
		m.markdownCache = strings.Split(out, "\n")
		m.markdownRenderCache = strings.Split(outRender, "\n")
		return outRender
	}
	outLines := strings.Split(out, "\n")
	prefixLines := findCommonPrefix(outLines, m.markdownCache)
	newOut := strings.Join(outLines[len(prefixLines):], "\n")
	newRender := Render(ThinkingStyle, width, newOut)
	m.markdownCache = append(m.markdownCache[:len(prefixLines)], strings.Split(newOut, "\n")...)
	m.markdownRenderCache = append(m.markdownRenderCache[:len(prefixLines)], strings.Split(newRender, "\n")...)
	return strings.Join(m.markdownRenderCache, "\n")
}
