package test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lgzzzz/gocode/internal/tui/compoent"
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

	fullRenderer := compoent.NewThinkingMessage("123", "")
	fullRenderer.SetFullRender(true)
	const width = 100
	t1 := time.Now()
	count := 0
	for i := 0; i <= len(content)-1; i = i + 10 {
		count++
		fullRenderer.SetContent(content[:i])
		fullRenderer.Render(width)
	}
	total := time.Since(t1)
	avg := total / time.Duration(count)
	fmt.Printf("总耗时: %s\n平均耗时: %s\n调用次数: %d\n", total.String(), avg.String(), count)

	//总耗时: 30.2409393s
	//平均耗时: 27.718551ms
	//调用次数: 1091
}

func TestRender_Increment(t *testing.T) {
	content := loadTestContent()
	if content == "" {
		t.Fatal("testMarkdown.md is empty")
	}

	incrementRenderer := compoent.NewThinkingMessage("123", "")
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

	//总耗时: 3.6788687s
	//平均耗时: 3.372015ms
	//调用次数: 1091
}
