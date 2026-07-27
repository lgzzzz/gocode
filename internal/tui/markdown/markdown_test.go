package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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

	fullRenderer := NewRenderer()
	fullRenderer.SetFullRender(true)
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
	fmt.Printf("全量绘制次数:%d\n增量绘制次数:%d\n部分绘制次数: %d\n部分绘制最大长度: %d\n", renderStat.fullRenderCount, renderStat.incrementRenderCount, renderStat.renderPartCount, renderStat.maxRenderContentLength)
	fmt.Println("部分绘制百分比:", float64(renderStat.maxRenderContentLength)/float64(len(content))*100, "%")
}

func TestRender_Increment(t *testing.T) {
	content := loadTestContent()
	if content == "" {
		t.Fatal("testMarkdown.md is empty")
	}

	incrementRenderer := NewRenderer()
	const width = 100
	t1 := time.Now()
	count := 0
	for i := 0; i <= len(content)-1; i = i + 10 {
		count++
		incrementRenderer.Render(content[:i], width)
	}
	total := time.Since(t1)
	avg := total / time.Duration(count)
	fmt.Printf("总耗时: %s\n平均耗时: %s\n调用次数: %d\n", total.String(), avg.String(), count)
	renderStat := incrementRenderer.Stat()
	fmt.Printf("全量绘制次数:%d\n增量绘制次数:%d\n部分绘制次数: %d\n部分绘制最大长度: %d\n", renderStat.fullRenderCount, renderStat.incrementRenderCount, renderStat.renderPartCount, renderStat.maxRenderContentLength)
	fmt.Println("部分绘制百分比:", float64(renderStat.maxRenderContentLength)/float64(len(content))*100, "%")
}
