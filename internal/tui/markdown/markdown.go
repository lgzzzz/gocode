package markdown

import (
	"strings"
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

// Renderer 是一个带流式优化缓存的 markdown 渲染器。
// 它缓存已渲染的"稳定前缀"，使得 AI 流式输出时只重新渲染尾部新增内容。
//
// 每个消息（AssistantMessage / ThinkingMessage）应持有自己的 Renderer 实例。
// 底层的 glamour.TermRenderer 按宽度全局共享，自动缓存。
type Renderer struct {
	lastWidth          int
	stablePrefix       string
	stablePrefixRender string
	split              splitDetector
}

// NewRenderer 创建一个 Renderer。
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Render 将 markdown 文本渲染为带 ANSI 样式的终端字符串。
// 流式场景下，如果 content 只是在上次渲染的基础上追加了内容，则只渲染新增部分。
func (r *Renderer) Render(content string, width int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	gr := getCachedRenderer(width)
	return r.render(content, width, gr)
}

// Reset 清空流式缓存。下次 Render 必定走全量渲染。
func (r *Renderer) Reset() {
	r.lastWidth = 0
	r.stablePrefix = ""
	r.stablePrefixRender = ""
}

// ---------------------------------------------------------------------------
// 流式渲染逻辑
// ---------------------------------------------------------------------------

func (r *Renderer) render(content string, width int, gr *glamour.TermRenderer) string {
	fullRender := func() string {
		out, err := gr.Render(content)
		if err != nil {
			return content
		}
		return strings.TrimSuffix(out, "\n")
	}

	if width != r.lastWidth || !strings.HasPrefix(content, r.stablePrefix) {
		r.Reset()
		r.lastWidth = width
		splitPoint := r.split.findSafeSplitPoint(content)
		r.stablePrefix = content[:splitPoint]
		stablePrefixRender, err := gr.Render(content[:splitPoint])
		if err != nil {
			return content
		}
		r.stablePrefixRender = stablePrefixRender
		render, err := gr.Render(content[splitPoint:])
		if err != nil {
			return content
		}
		return joinParts(stablePrefixRender, render)
	}

	splitPoint := r.split.findSafeSplitPoint(content)
	if splitPoint < 0 {
		return fullRender()
	}

	if splitPoint <= len(r.stablePrefix) {
		newPart := content[len(r.stablePrefix):]
		return joinParts(r.stablePrefixRender, r.renderPart(newPart, gr))
	}

	newSafe := content[len(r.stablePrefix):splitPoint]
	newSafeRender := r.renderPart(newSafe, gr)
	r.stablePrefixRender = joinParts(r.stablePrefixRender, newSafeRender)
	r.stablePrefix = content[:splitPoint]

	remainder := content[splitPoint:]
	if remainder == "" {
		return r.stablePrefixRender
	}
	return joinParts(r.stablePrefixRender, r.renderPart(remainder, gr))
}

func (r *Renderer) renderPart(text string, gr *glamour.TermRenderer) string {
	if text == "" {
		return ""
	}
	out, err := gr.Render(text)
	if err != nil {
		return text
	}
	return strings.TrimSpace(out)
}

func joinParts(a, b string) string {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	switch {
	case a == "" && b == "":
		return ""
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "\n\n" + b
	}
}

// ============================================================================
// glamour 渲染器缓存
// ============================================================================

var (
	mdCacheMu sync.Mutex
	mdCache   = map[int]*glamour.TermRenderer{}
)

func getCachedRenderer(width int) *glamour.TermRenderer {
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

// InvalidateCache 清空底层 glamour 渲染器缓存。主题样式变化时调用。
func InvalidateCache() {
	mdCacheMu.Lock()
	defer mdCacheMu.Unlock()
	mdCache = map[int]*glamour.TermRenderer{}
}
