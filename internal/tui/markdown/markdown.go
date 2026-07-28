package markdown

import (
	"strings"
	"sync"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/lgzzzz/gocode/internal/tui/util"
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

type RenderStaus struct {
	FullRenderCount        int
	IncrementRenderCount   int
	RenderPartCount        int
	MaxRenderContentLength int
}

// Renderer 是一个带流式优化缓存的 markdown 渲染器。
// 它缓存已渲染的"稳定前缀"，使得 AI 流式输出时只重新渲染尾部新增内容。
//
// 每个消息（AssistantMessage / ThinkingMessage）应持有自己的 Renderer 实例。
// 底层的 glamour.TermRenderer 按宽度全局共享，自动缓存。
type Renderer struct {
	contentWidth       int // 内容区宽度（不含左侧 border + padding）
	displayWidth       int // 展示区宽度（含左侧 border + padding）
	lastContentWidth   int // 上次渲染的内容区宽度，用于检测宽度变化
	stablePrefix       string
	stablePrefixRender string
	split              splitDetector
	fullRender         bool

	style lipgloss.Style
	gr    *glamour.TermRenderer

	renderStatus RenderStaus
}

// NewRenderer 创建一个 Renderer。
// style 会作用于 glamour 渲染之后的每一部分，并缓存在 stablePrefixRender 中。
func NewRenderer(style lipgloss.Style) *Renderer {
	return &Renderer{style: style}
}

func (r *Renderer) SetFullRender(fullRender bool) {
	r.fullRender = fullRender
}

// Render 将 markdown 文本渲染为带 ANSI 样式的终端字符串。
// 流式场景下，如果 content 只是在上次渲染的基础上追加了内容，则只渲染新增部分。
func (r *Renderer) Render(content string, width int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	r.contentWidth = width - 2
	r.displayWidth = width
	r.ensureGR()
	return r.render(content)
}

// ensureGR 确保 r.gr 为当前 contentWidth 对应的 glamour 渲染器。
func (r *Renderer) ensureGR() {
	if r.gr == nil || r.contentWidth != r.lastContentWidth {
		r.gr = getCachedRenderer(r.contentWidth)
	}
}

// Reset 清空流式缓存。下次 Render 必定走全量渲染。
func (r *Renderer) Reset() {
	r.lastContentWidth = 0
	r.stablePrefix = ""
	r.stablePrefixRender = ""
}

func (r *Renderer) Stat() RenderStaus {
	return r.renderStatus
}

// ---------------------------------------------------------------------------
// 流式渲染逻辑
// ---------------------------------------------------------------------------

func (r *Renderer) render(content string) string {
	styled := r.style.Width(r.displayWidth)

	fullRender := func() string {
		r.renderStatus.FullRenderCount++
		out, err := r.gr.Render(content)
		if err != nil {
			return styled.Render(util.TrimEmptyLine(content))
		}
		return styled.Render(util.TrimEmptyLine(out))
	}
	if r.fullRender {
		return fullRender()
	}

	if r.contentWidth != r.lastContentWidth || !strings.HasPrefix(content, r.stablePrefix) {
		r.Reset()
		r.lastContentWidth = r.contentWidth
		out := fullRender()
		r.tryCachePrefix(content)
		return out
	}

	r.renderStatus.IncrementRenderCount++
	splitPoint := r.split.findSafeSplitPoint(content)
	if splitPoint < 0 {
		return fullRender()
	}

	if splitPoint <= len(r.stablePrefix) {
		newPart := content[len(r.stablePrefix):]
		newPartStyled := r.renderPart(newPart)
		return r.joinParts(r.stablePrefixRender, newPartStyled)
	}

	newSafe := content[len(r.stablePrefix):splitPoint]
	newSafeStyled := r.renderPart(newSafe)
	r.stablePrefixRender = r.joinParts(r.stablePrefixRender, newSafeStyled)
	r.stablePrefix = content[:splitPoint]

	remainder := content[splitPoint:]
	if remainder == "" {
		return r.stablePrefixRender
	}
	remainderStyled := r.renderPart(remainder)
	return r.joinParts(r.stablePrefixRender, remainderStyled)
}

func (r *Renderer) renderPart(text string) string {
	r.renderStatus.RenderPartCount++
	if len(text) > r.renderStatus.MaxRenderContentLength {
		r.renderStatus.MaxRenderContentLength = len(text)
	}
	out, err := r.gr.Render(text)
	if err != nil {
		text = util.TrimEmptyLine(text)
		if text == "" {
			return ""
		}
		return r.style.Width(r.displayWidth).Render(text)
	}
	out = util.TrimEmptyLine(out)
	if out == "" {
		return ""
	}
	return r.style.Width(r.displayWidth).Render(out)
}

func (r *Renderer) tryCachePrefix(content string) {
	r.renderStatus.FullRenderCount++
	splitPoint := r.split.findSafeSplitPoint(content)
	if splitPoint <= 0 {
		return
	}
	prefix := content[:splitPoint]
	out, err := r.gr.Render(prefix)
	if err != nil {
		return
	}
	styled := r.style.Width(r.displayWidth)
	r.stablePrefix = prefix
	r.stablePrefixRender = styled.Render(util.TrimEmptyLine(out))
}

// joinParts 将两段已应用 lipgloss style 的渲染结果拼接。
// 使用单个 \n 而非 \n\n，因为每行已带有左侧边框，空行会破坏边框连续性。
func (r *Renderer) joinParts(a, b string) string {
	switch {
	case a == "" && b == "":
		return ""
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "\n" + r.style.Render(" ") + "\n" + b
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
