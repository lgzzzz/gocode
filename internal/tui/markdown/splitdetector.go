package markdown

import "strings"

// splitDetector 负责在 markdown 文本中寻找安全切分点。
// 安全切分点是一个空行之后的位置，且该位置之前没有未闭合的 markdown 结构。
type splitDetector struct{}

// findSafeSplitPoint 从后往前找第一个安全的切分位置。
// 返回 content[:offset] 可作为稳定前缀的偏移量，找不到返回 -1。
func (d *splitDetector) findSafeSplitPoint(content string) int {
	if len(content) == 0 {
		return -1
	}
	for p := d.lastBlankEnd(content, len(content)); p > 0; p = d.lastBlankEnd(content, p-1) {
		if d.canSplitAt(content, p) {
			return p
		}
	}
	return -1
}

// lastBlankEnd 返回 until 之前最近的空行结束位置。
func (d *splitDetector) lastBlankEnd(content string, until int) int {
	if until <= 0 {
		return -1
	}
	end := until
	for end > 0 {
		nl := strings.LastIndexByte(content[:end], '\n')
		if nl < 0 {
			return -1
		}
		prev := strings.LastIndexByte(content[:nl], '\n')
		if prev >= 0 {
			if strings.TrimSpace(content[prev+1:nl]) == "" {
				return nl + 1
			}
		}
		end = nl
	}
	return -1
}

// canSplitAt 判断 content[:p] 是否可以作为安全前缀独立渲染。
func (d *splitDetector) canSplitAt(content string, p int) bool {
	prefix := content[:p]
	if d.countFences(prefix)%2 != 0 {
		return false
	}
	return true
}

// countFences 统计围栏代码块标记行的数量。偶数表示所有代码块已闭合。
func (d *splitDetector) countFences(s string) int {
	n := 0
	lines := strings.SplitSeq(s, "\n")
	for line := range lines {
		if d.isFence(line) {
			n++
		}
	}
	return n
}

// isFence 判断一行是否是围栏代码块标记（``` 或 ~~~）。
func (d *splitDetector) isFence(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}
