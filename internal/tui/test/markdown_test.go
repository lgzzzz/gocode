package test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lgzzzz/gocode/internal/tui/compoent"
	"github.com/lgzzzz/gocode/internal/tui/markdown"
)

// ============================================================================
// 公共测试内容
// ============================================================================

// loadTestContent 加载 testMarkdown.md 的全部内容作为测试输入。
func loadTestContent() string {
	// 获取当前文件所在目录，兼容从不同目录运行测试。
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	data, err := os.ReadFile(filepath.Join(dir, "testMarkdown.md"))
	if err != nil {
		panic(fmt.Sprintf("failed to load testMarkdown.md: %v", err))
	}
	return string(data)
}

// TestRender_Comparison 同时运行全量和增量渲染，输出对比结果。
func TestRender_FullRender(t *testing.T) {
	content := loadTestContent()
	if content == "" {
		t.Fatal("testMarkdown.md is empty")
	}

	fullRenderer := markdown.NewRenderer(compoent.AssistantStyle)
	const width = 100
	t1 := time.Now()
	count := 0
	for i := 0; i <= len(content)-1; i = i + 10 {
		count++
		fullRenderer.Render(content[:i], width)
	}
	total := time.Since(t1)
	avg := total / time.Duration(count)
	fmt.Printf("总耗时: %s\n平均耗时: %s\n调用次数: %d\n", total.String(), avg.String(), count)
	renderStat := fullRenderer.Stat()
	fmt.Printf("全量绘制次数:%d\n增量绘制次数:%d\n部分绘制次数: %d\n部分绘制最大长度: %d\n", renderStat.FullRenderCount, renderStat.IncrementRenderCount, renderStat.RenderPartCount, renderStat.MaxRenderContentLength)
	fmt.Println("部分绘制百分比:", float64(renderStat.MaxRenderContentLength)/float64(len(content))*100, "%")

	// 总耗时: 28.9044823s
	// 平均耗时: 26.988312ms
}

func TestRender_Increment(t *testing.T) {
	content := loadTestContent()
	if content == "" {
		t.Fatal("testMarkdown.md is empty")
	}

	incrementRenderer := compoent.NewAssistantMessage("1", "")
	const width = 100
	t1 := time.Now()
	count := 0
	for i := 0; i <= len(content)-1; i = i + 10 {
		count++
		incrementRenderer.SetContent(content[:i])
		incrementRenderer.Render(width)
	}
	total := time.Since(t1)
	avg := total / time.Duration(count)
	fmt.Printf("总耗时: %s\n平均耗时: %s\n调用次数: %d\n", total.String(), avg.String(), count)

	// 总耗时: 10.367235s
	// 平均耗时: 9.679957ms

	// 总耗时: 2.9922679s
	// 平均耗时: 2.7939ms
}
