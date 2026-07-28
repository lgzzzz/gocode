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

func TestRender_NoCacheRender(t *testing.T) {
	content := loadTestContent()
	if content == "" {
		t.Fatal("testMarkdown.md is empty")
	}

	render := compoent.NewThinkingMessage("123", "")
	render.SetFullRender(true)
	const width = 100
	t1 := time.Now()
	count := 0
	for i := 0; i <= len(content)-1; i = i + 10 {
		count++
		render.SetContent(content[:i])
		render.Render(width)
	}
	total := time.Since(t1)
	avg := total / time.Duration(count)
	fmt.Printf("总耗时: %s\n平均耗时: %s\n调用次数: %d\n", total.String(), avg.String(), count)

	//=== RUN   TestRender_NoCacheRender
	//总耗时: 30.4784415s
	//平均耗时: 27.936243ms
	//调用次数: 1091
	//--- PASS: TestRender_NoCacheRender (30.48s)
	//PASS
}

func TestRender_MarkdownCacheRender(t *testing.T) {
	content := loadTestContent()
	if content == "" {
		t.Fatal("testMarkdown.md is empty")
	}

	render := compoent.NewThinkingMessage("123", "")
	render.SetFullStyleRender(true)
	const width = 100
	t1 := time.Now()
	count := 0
	for i := 0; i <= len(content)-1; i = i + 10 {
		count++
		render.SetContent(content[:i])
		render.Render(width)
	}
	total := time.Since(t1)
	avg := total / time.Duration(count)
	fmt.Printf("总耗时: %s\n平均耗时: %s\n调用次数: %d\n", total.String(), avg.String(), count)

	//=== RUN   TestRender_MarkdownCacheRender
	//总耗时: 10.940981s
	//平均耗时: 10.028396ms
	//调用次数: 1091
	//--- PASS: TestRender_MarkdownCacheRender (10.94s)
	//PASS
}

func TestRender_FullCacheRender(t *testing.T) {
	content := loadTestContent()
	if content == "" {
		t.Fatal("testMarkdown.md is empty")
	}

	render := compoent.NewThinkingMessage("123", "")
	const width = 100
	t1 := time.Now()
	count := 0
	for i := 0; i <= len(content)-1; i = i + 10 {
		count++
		render.SetContent(content[:i])
		render.Render(width)
	}
	total := time.Since(t1)
	avg := total / time.Duration(count)
	fmt.Printf("总耗时: %s\n平均耗时: %s\n调用次数: %d\n", total.String(), avg.String(), count)

	//=== RUN   TestRender_FullCacheRender
	//总耗时: 3.6705642s
	//平均耗时: 3.364403ms
	//调用次数: 1091
	//--- PASS: TestRender_FullCacheRender (3.67s)
	//PASS
}
