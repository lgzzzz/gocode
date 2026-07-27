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
		for prev >= 0 {
			if d.isEmptyLine(content[prev+1 : nl]) {
				return nl + 1
			}
			break
		}
		end = nl
	}
	return -1
}

// isEmptyLine 判断字符串是否为空或仅由空格/制表符组成。
func (d *splitDetector) isEmptyLine(s string) bool {
	for i := range len(s) {
		if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return true
}

// canSplitAt 判断 content[:p] 是否可以作为安全前缀独立渲染。
func (d *splitDetector) canSplitAt(content string, p int) bool {
	prefix := content[:p]
	if d.countFences(prefix)%2 != 0 {
		return false
	}
	//if d.hasOpenBlock(prefix) {
	//	return false
	//}
	//if last := d.lastNonBlank(prefix); last != "" && d.isBlockLine(last) {
	//	return false
	//}
	//if rest := content[p:]; rest != "" {
	//	if first := d.firstNonBlank(rest); d.isSetextLine(first) {
	//		return false
	//	}
	//}
	return true
}

// hasOpenBlock 检查前缀中是否存在无法在空行边界安全切分的结构：
// 松散列表、HTML 块、链接引用定义。
func (d *splitDetector) hasOpenBlock(prefix string) bool {
	inFence := false
	for line := range d.lines(prefix) {
		if d.isFence(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		t := strings.TrimLeft(line, " \t")
		if t == "" {
			continue
		}
		if d.isListItem(t) || d.isHTMLStart(line) || d.isRefDef(line) {
			return true
		}
	}
	return false
}

// countFences 统计围栏代码块标记行的数量。偶数表示所有代码块已闭合。
func (d *splitDetector) countFences(s string) int {
	n := 0
	for line := range d.lines(s) {
		if d.isFence(line) {
			n++
		}
	}
	return n
}

// isFence 判断一行是否是围栏代码块标记（``` 或 ~~~）。
func (d *splitDetector) isFence(line string) bool {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) {
		return false
	}
	c := line[i]
	if c != '`' && c != '~' {
		return false
	}
	n := 0
	for i < len(line) && line[i] == c {
		i++
		n++
	}
	return n >= 3
}

// lastNonBlank 返回最后一行非空内容。
func (d *splitDetector) lastNonBlank(s string) string {
	last := ""
	for line := range d.lines(s) {
		if strings.TrimSpace(line) != "" {
			last = line
		}
	}
	return last
}

// firstNonBlank 返回第一行非空内容。
func (d *splitDetector) firstNonBlank(s string) string {
	for line := range d.lines(s) {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
	return ""
}

// lines 按换行拆分字符串，返回迭代器。
func (d *splitDetector) lines(s string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		start := 0
		for i := 0; i < len(s); i++ {
			if s[i] == '\n' {
				if !yield(s[start:i]) {
					return
				}
				start = i + 1
			}
		}
		if start <= len(s)-1 {
			yield(s[start:])
		}
	}
}

// isBlockLine 判断一行是否开启或延续一个块级结构。
func (d *splitDetector) isBlockLine(line string) bool {
	if len(line) > 0 && line[0] == '\t' {
		return true
	}
	if strings.HasPrefix(line, "    ") {
		return true
	}
	t := strings.TrimLeft(line, " \t")
	if t == "" {
		return false
	}
	if t[0] == '>' || d.isListItem(t) || strings.ContainsRune(line, '|') || d.isSetextLine(t) {
		return true
	}
	return false
}

// isListItem 判断行首是否是列表标记（- 、* 、+ 、数字. 、数字) ）。
func (d *splitDetector) isListItem(line string) bool {
	if line == "" {
		return false
	}
	c := line[0]
	if c == '-' || c == '*' || c == '+' {
		return len(line) >= 2 && (line[1] == ' ' || line[1] == '\t')
	}
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i > 9 || i >= len(line) {
		return false
	}
	if line[i] != '.' && line[i] != ')' {
		return false
	}
	return i+1 < len(line) && (line[i+1] == ' ' || line[i+1] == '\t')
}

// isSetextLine 判断一行是否是 setext 标题的下划线（=== 或 ---）。
func (d *splitDetector) isSetextLine(line string) bool {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	if i == len(line) {
		return false
	}
	c := line[i]
	if c != '=' && c != '-' {
		return false
	}
	j := i
	for j < len(line) && line[j] == c {
		j++
	}
	for j < len(line) {
		if line[j] != ' ' && line[j] != '\t' {
			return false
		}
		j++
	}
	return j-i >= 1
}

// isHTMLStart 判断一行是否是 HTML 块的开始（CommonMark 7 种模式）。
func (d *splitDetector) isHTMLStart(line string) bool {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	rest := line[i:]
	if len(rest) < 2 || rest[0] != '<' {
		return false
	}
	if strings.HasPrefix(rest, "<!--") || strings.HasPrefix(rest, "<?") || strings.HasPrefix(rest, "<![CDATA[") {
		return true
	}
	if len(rest) >= 3 && rest[1] == '!' && d.isASCIILetter(rest[2]) {
		return true
	}
	low := strings.ToLower(rest)
	for _, tag := range []string{"<script", "<pre", "<style", "<textarea"} {
		if strings.HasPrefix(low, tag) {
			next := byte(0)
			if len(low) > len(tag) {
				next = low[len(tag)]
			}
			if next == 0 || next == ' ' || next == '\t' || next == '>' {
				return true
			}
		}
	}
	j := 1
	if j < len(rest) && rest[j] == '/' {
		j++
	}
	return j < len(rest) && d.isASCIILetter(rest[j])
}

// isASCIILetter 判断字节是否是 ASCII 字母。
func (d *splitDetector) isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isRefDef 判断一行是否是链接引用定义 [label]: url。
func (d *splitDetector) isRefDef(line string) bool {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) || line[i] != '[' {
		return false
	}
	i++
	start := i
	for i < len(line) && line[i] != ']' {
		i++
	}
	if i >= len(line) || i == start {
		return false
	}
	i++
	if i >= len(line) || line[i] != ':' {
		return false
	}
	i++
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return i < len(line)
}
