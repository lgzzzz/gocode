package markdown

import (
	"regexp"
	"strings"
	"sync"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func isEmptyOrInvisible(s string) bool {
	return strings.TrimSpace(ansiRe.ReplaceAllString(s, "")) == ""
}

func trimEmptyLine(content string) string {
	// Trim leading empty lines
	for {
		index := strings.Index(content, "\n")
		if index == -1 {
			break
		}
		if isEmptyOrInvisible(content[:index]) {
			content = content[index+1:]
		} else {
			break
		}
	}
	// Trim trailing empty lines
	for {
		index := strings.LastIndex(content, "\n")
		if index == -1 {
			break
		}
		if isEmptyOrInvisible(content[index+1:]) {
			content = content[:index]
		} else {
			break
		}
	}
	return content
}

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
		return trimEmptyLine(out)
	}

	if width != r.lastWidth || !strings.HasPrefix(content, r.stablePrefix) {
		r.Reset()
		r.lastWidth = width
		out := fullRender()
		r.tryCachePrefix(content, width, gr)
		return out
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
	out, err := gr.Render(text)
	if err != nil {
		return text
	}
	return out
}

func (r *Renderer) tryCachePrefix(content string, width int, gr *glamour.TermRenderer) {
	splitPoint := r.split.findSafeSplitPoint(content)
	if splitPoint <= 0 {
		return
	}
	prefix := content[:splitPoint]
	out, err := gr.Render(prefix)
	if err != nil {
		return
	}
	r.stablePrefix = prefix
	r.stablePrefixRender = out
	r.lastWidth = width
}

func joinParts(a, b string) string {
	a = trimEmptyLine(a)
	b = trimEmptyLine(b)
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
