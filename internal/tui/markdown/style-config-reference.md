# Glamour ANSI StyleConfig 配置参考

本文档详细说明了 `charm.land/glamour/v2/ansi` 包中 `StyleConfig` 及其相关类型的各个配置项。

---

## 概述

`StyleConfig` 用于配置 `ANSIRenderer` 的样式行为，控制 Markdown 转换到终端 ANSI 输出时的渲染效果。它定义了文档中每个 Markdown 元素的颜色、文本样式（粗体、斜体等）、前缀/后缀、缩进等。

---

## 核心类型

### `StylePrimitive` — 基础样式单元

所有文本样式的基础结构，被 `StyleBlock` 及其他样式类型嵌入。

| 字段 | 类型 | 说明 |
|------|------|------|
| `BlockPrefix` | `string` | 块级元素整体前的前缀字符串（如列表项前的 `"• "`） |
| `BlockSuffix` | `string` | 块级元素整体后的后缀字符串 |
| `Prefix` | `string` | 元素内容前的前缀文本（不可被样式继承覆盖） |
| `Suffix` | `string` | 元素内容后的后缀文本（不可被样式继承覆盖） |
| `Color` | `*string` | 前景色。支持 ANSI 颜色号（`"39"`）、256色（`"203"`）、或十六进制（`"#C4C4C4"`） |
| `BackgroundColor` | `*string` | 背景色，格式同 `Color` |
| `Underline` | `*bool` | 是否添加下划线 |
| `Bold` | `*bool` | 是否粗体 |
| `Upper` | `*bool` | 是否转换为大写 |
| `Lower` | `*bool` | 是否转换为小写 |
| `Title` | `*bool` | 是否转换为标题大小写（每个单词首字母大写） |
| `Italic` | `*bool` | 是否斜体 |
| `CrossedOut` | `*bool` | 是否添加删除线 |
| `Faint` | `*bool` | 是否使用淡色/暗色输出 |
| `Conceal` | `*bool` | 是否隐藏文本 |
| `Inverse` | `*bool` | 是否反转前景色和背景色 |
| `Blink` | `*bool` | 是否闪烁文本 |
| `Format` | `string` | Go 模板字符串，用于自定义格式化输出。例如 `"Image: {{.text}} →"` 会使用模板渲染 |

---

### `StyleBlock` — 块级元素样式

嵌入 `StylePrimitive` 并添加块级布局相关的属性。用于段落、标题、引用、代码块等。

| 字段 | 类型 | 说明 |
|------|------|------|
| `StylePrimitive` | (嵌入) | 继承所有基础样式属性 |
| `Indent` | `*uint` | 缩进级别（空格数） |
| `IndentToken` | `*string` | 缩进时行的前缀标记。例如 `"│ "` 会在每行前添加竖线和空格 |
| `Margin` | `*uint` | 块的外边距（上下空行数） |

---

### `StyleList` — 列表样式

嵌入 `StyleBlock` 并添加列表特有的层级缩进。

| 字段 | 类型 | 说明 |
|------|------|------|
| `StyleBlock` | (嵌入) | 继承块级样式属性 |
| `LevelIndent` | `uint` | 每级嵌套列表的额外缩进空格数 |

---

### `StyleTask` — 任务列表样式

嵌入 `StylePrimitive` 并自定义勾选/未勾选的标记。

| 字段 | 类型 | 说明 |
|------|------|------|
| `StylePrimitive` | (嵌入) | 继承基础样式属性 |
| `Ticked` | `string` | 已完成任务的标记，默认 `"[✓] "` |
| `Unticked` | `string` | 未完成任务的标记，默认 `"[ ] "` |

---

### `StyleTable` — 表格样式

嵌入 `StyleBlock` 并自定义表格分隔线字符。

| 字段 | 类型 | 说明 |
|------|------|------|
| `StyleBlock` | (嵌入) | 继承块级样式属性 |
| `CenterSeparator` | `*string` | 表头与表体之间的分隔线字符，默认 `"┼"` |
| `ColumnSeparator` | `*string` | 列分隔符，默认 `"│"` |
| `RowSeparator` | `*string` | 行分隔线使用的水平字符，默认 `"─"` |

---

### `StyleCodeBlock` — 代码块样式

嵌入 `StyleBlock` 并支持语法高亮主题。

| 字段 | 类型 | 说明 |
|------|------|------|
| `StyleBlock` | (嵌入) | 继承块级样式属性 |
| `Theme` | `string` | 预定义主题名称。若设置了 `Chroma` 则被忽略 |
| `Chroma` | `*Chroma` | 自定义 Chroma 语法高亮颜色配置（见下方） |

---

### `Chroma` — 语法高亮颜色

与 [Chroma](https://github.com/alecthomas/chroma) 语法高亮库的 token 类型一一对应。每个字段都是一个 `StylePrimitive`，用于自定义该 token 类型的前景色/背景色等。

| 字段 | 对应的语法 Token | 说明 |
|------|-------------------|------|
| `Text` | `Text` | 普通文本 |
| `Error` | `Error` | 词法/语法错误 |
| `Comment` | `Comment` | 注释 |
| `CommentPreproc` | `CommentPreproc` | 预处理器指令（如 C 的 `#include`） |
| `Keyword` | `Keyword` | 关键字（通用） |
| `KeywordReserved` | `KeywordReserved` | 保留关键字 |
| `KeywordNamespace` | `KeywordNamespace` | 命名空间关键字 |
| `KeywordType` | `KeywordType` | 类型关键字（如 `int`, `string`） |
| `Operator` | `Operator` | 操作符（如 `+`, `=`, `->`） |
| `Punctuation` | `Punctuation` | 标点符号（如 `;`, `,`, `{}`） |
| `Name` | `Name` | 标识符/名称（通用） |
| `NameBuiltin` | `NameBuiltin` | 内建名称（如 `print`, `len`） |
| `NameTag` | `NameTag` | 标签（如 HTML 标签） |
| `NameAttribute` | `NameAttribute` | 属性名（如 HTML 属性） |
| `NameClass` | `NameClass` | 类名 |
| `NameConstant` | `NameConstant` | 常量名 |
| `NameDecorator` | `NameDecorator` | 装饰器/注解名 |
| `NameException` | `NameException` | 异常名 |
| `NameFunction` | `NameFunction` | 函数名 |
| `NameOther` | `NameOther` | 其他名称 |
| `Literal` | `Literal` | 字面量（通用） |
| `LiteralNumber` | `LiteralNumber` | 数字字面量（如 `42`, `3.14`） |
| `LiteralDate` | `LiteralDate` | 日期字面量 |
| `LiteralString` | `LiteralString` | 字符串字面量 |
| `LiteralStringEscape` | `LiteralStringEscape` | 字符串转义序列（如 `\n`） |
| `GenericDeleted` | `GenericDeleted` | diff 中的删除行 |
| `GenericEmph` | `GenericEmph` | 强调（斜体） |
| `GenericInserted` | `GenericInserted` | diff 中的插入行 |
| `GenericStrong` | `GenericStrong` | 加强（粗体） |
| `GenericSubheading` | `GenericSubheading` | 副标题 |
| `Background` | `Background` | 代码块整体背景色 |

---

## `StyleConfig` 配置项详解

### 文档结构

#### `Document`
- **类型**: `StyleBlock`
- **说明**: 整个文档的外层容器样式。
- **常用属性**: `Margin`（文档与终端边缘的距离）、`Color`（默认文本色）。

#### `Paragraph`
- **类型**: `StyleBlock`
- **说明**: 普通段落 `<p>` 的样式。

---

### 引用块

#### `BlockQuote`
- **类型**: `StyleBlock`
- **说明**: Markdown 引用块 `>` 的样式。
- **常用配置**:
  - `Indent`: 引用内容缩进。
  - `IndentToken`: 每行前的引用装饰线，如 `"│ "`。

---

### 列表

#### `List`
- **类型**: `StyleList`
- **说明**: 无序列表 `<ul>` / 有序列表 `<ol>` 的整体样式。
- **常用配置**:
  - `LevelIndent`: 嵌套列表层级缩进量。

#### `Item`
- **类型**: `StylePrimitive`
- **说明**: 无序列表项 `<li>` 的样式。
- **常用配置**: `BlockPrefix`（如 `"• "` 设置项目符号）。

#### `Enumeration`
- **类型**: `StylePrimitive`
- **说明**: 有序列表项 `<li>` 的样式。
- **常用配置**: `BlockPrefix`（如 `". "`，序号会自动拼接为 `"1. "`）。

#### `Task`
- **类型**: `StyleTask`
- **说明**: 任务列表 `- [ ]` / `- [x]` 的样式。
- **常用配置**: `Ticked`（已完成标记）、`Unticked`（未完成标记）。

---

### 标题

#### `Heading`
- **类型**: `StyleBlock`
- **说明**: 所有标题（H1-H6）的**通用基础样式**。每个具体标题级别（H1-H6）会在此基础上叠加自身的配置。

- **常用配置**:
  - `Color`: 标题文本色。
  - `Bold`: 是否粗体。
  - `BlockSuffix`: 标题后的空行（如 `"\n"`）。

#### `H1` — `H6`
- **类型**: `StyleBlock`
- **说明**: 各级标题的具体样式，会与 `Heading` 的样式合并。
- **常用配置**:
  - `Prefix`: 标题前文本（如 `"## "` 表示 H2 以 `## ` 开头）。
  - `Color` / `BackgroundColor`: H1 常用反显（彩色背景）突出。
  - `Bold`: H1 通常加粗。

> **样式合并规则**: `Heading` 定义通用基础，`H1`-`H6` 定义各自的差异化属性。最终渲染时两者合并，子级可覆盖父级属性。

---

### 文本内联元素

#### `Text`
- **类型**: `StylePrimitive`
- **说明**: 普通文本的默认样式。

#### `Strikethrough`
- **类型**: `StylePrimitive`
- **说明**: 删除线 `~~text~~` 的样式。
- **常用配置**: `CrossedOut`。

#### `Emph`
- **类型**: `StylePrimitive`
- **说明**: 斜体强调 `*text*` / `_text_` 的样式。
- **常用配置**: `Italic`。

#### `Strong`
- **类型**: `StylePrimitive`
- **说明**: 粗体强调 `**text**` / `__text__` 的样式。
- **常用配置**: `Bold`。

#### `HorizontalRule`
- **类型**: `StylePrimitive`
- **说明**: 水平分割线 `---` / `***` 的样式。

---

### 链接

#### `Link`
- **类型**: `StylePrimitive`
- **说明**: 链接 URL 部分的样式（`[text](url)` 中的 `url` 渲染）。
- **常用配置**: `Color`, `Underline`。

#### `LinkText`
- **类型**: `StylePrimitive`
- **说明**: 链接文本部分的样式（`[text](url)` 中的 `text` 渲染）。
- **常用配置**: `Color`, `Bold`。

---

### 图片

#### `Image`
- **类型**: `StylePrimitive`
- **说明**: 图片 URL 部分的样式。
- **常用配置**: `Color`, `Underline`。

#### `ImageText`
- **类型**: `StylePrimitive`
- **说明**: 图片替代文本的样式。
- **常用配置**: `Format` 可自定义模板，如 `"Image: {{.text}} →"` 会将图片渲染为自定义格式。

---

### 代码

#### `Code`
- **类型**: `StyleBlock`
- **说明**: 行内代码 `` `code` `` 的样式。
- **常用配置**:
  - `Color`: 代码文本色。
  - `BackgroundColor`: 代码背景色。
  - `Prefix` / `Suffix`: 内边距（如 `"\u00a0"` 不间断空格）。

#### `CodeBlock`
- **类型**: `StyleCodeBlock`
- **说明**: 围栏代码块 ` ``` ` 的样式。
- **常用配置**:
  - `Margin`: 代码块上下外边距。
  - `Theme` 或 `Chroma`: 语法高亮方案。`Chroma` 提供更精细的 token 级控制。

---

### 表格

#### `Table`
- **类型**: `StyleTable`
- **说明**: Markdown 表格的样式。
- **常用配置**:
  - `Margin`: 表格外边距。
  - `CenterSeparator`: 表体行间的分隔线符。
  - `ColumnSeparator`: 列间分隔符。
  - `RowSeparator`: 表头与表体分隔线的填充字符。

---

### 定义列表

#### `DefinitionList`
- **类型**: `StyleBlock`
- **说明**: 定义列表 `<dl>` 的整体样式。

#### `DefinitionTerm`
- **类型**: `StylePrimitive`
- **说明**: 定义术语 `<dt>` 的样式。

#### `DefinitionDescription`
- **类型**: `StylePrimitive`
- **说明**: 定义描述 `<dd>` 的样式。
- **常用配置**: `BlockPrefix`（如 `"\n🠶 "` 在每个描述前添加前缀）。

---

### HTML

#### `HTMLBlock`
- **类型**: `StyleBlock`
- **说明**: 块级 HTML 元素（如 `<div>`）的样式。

#### `HTMLSpan`
- **类型**: `StyleBlock`
- **说明**: 内联 HTML 元素（如 `<span>`）的样式。

---

## `darkCompactConfig` 样例解析

以下是用户代码中 `darkCompactConfig` 的关键设计要点：

| 配置路径 | 值 | 设计意图 |
|----------|-----|----------|
| `Document.Margin` | `2` | 文档与终端边缘保持 2 行间距 |
| `Document.Color` | `"252"` | 默认文本为浅灰色（256色） |
| `BlockQuote.IndentToken` | `"│ "` | 引用块左侧用竖线装饰 |
| `List.LevelIndent` | `2` | 嵌套列表每级缩进 2 个空格 |
| `Heading.Color` | `"39"` | 标题默认蓝色 |
| `Heading.Bold` | `true` | 所有标题默认粗体 |
| `Heading.BlockSuffix` | `"\n"` | 标题后跟一个空行 |
| `H1.Color` | `"228"` | H1 文本为亮黄色 |
| `H1.BackgroundColor` | `"63"` | H1 背景为深蓝色（反显效果） |
| `H1.Prefix` / `H1.Suffix` | `" "` | H1 左右各留一个空格内边距 |
| `H2.Prefix` | `"## "` | H2 前渲染 `## ` 标记 |
| `H6.Color` | `"35"` | H6 使用紫色，且不加粗 |
| `Strikethrough.CrossedOut` | `true` | 删除线文本显示删除线 |
| `Emph.Italic` | `true` | 斜体渲染 |
| `Strong.Bold` | `true` | 粗体渲染 |
| `Item.BlockPrefix` | `"• "` | 无序列表用圆点 |
| `Enumeration.BlockPrefix` | `". "` | 有序列表用 `1. ` 格式 |
| `Task.Ticked` | `"[✓] "` | 已完成任务显示勾号 |
| `Task.Unticked` | `"[ ] "` | 未完成任务显示空格 |
| `Link.Color` | `"30"` | 链接 URL 用暗色 |
| `Link.Underline` | `true` | 链接 URL 加下划线 |
| `LinkText.Color` | `"35"` | 链接文本用紫色 |
| `LinkText.Bold` | `true` | 链接文本粗体 |
| `Code.Color` | `"203"` | 行内代码红色文本 |
| `Code.BackgroundColor` | `"236"` | 行内代码深灰背景 |
| `Code.Prefix` / `Code.Suffix` | `"\u00a0"` | 行内代码用不间断空格做内边距 |
| `CodeBlock.Margin` | `2` | 代码块上下各 2 行间距 |
| `Table.Margin` | `0` | 表格紧凑无外边距 |
| `Table.CenterSeparator` | `"┼"` | 表格行间用 `┼` 分隔 |
| `Table.ColumnSeparator` | `"│"` | 列间用竖线 `│` |
| `Table.RowSeparator` | `"─"` | 表头分隔线用 `─` |
| `DefinitionDescription.BlockPrefix` | `"\n🠶 "` | 定义描述前换行加箭头前缀 |

### Chroma 语法高亮配色方案

整体为**暗色主题**（`Background` 背景色为 `#373737` 深灰），特点：

- **关键字** (`Keyword`): 亮蓝 `#00AAFF`
- **保留关键字** (`KeywordReserved`): 亮粉 `#FF5FD2`
- **类型关键字** (`KeywordType`): 紫蓝 `#6E6ED8`
- **函数名** (`NameFunction`): 绿色 `#00D787`
- **类名** (`NameClass`): 白色粗体带下划线，突出显示
- **字符串** (`LiteralString`): 暖棕 `#C69669`
- **数字** (`LiteralNumber`): 薄荷绿 `#6EEFC0`
- **注释** (`Comment`): 暗灰 `#676767`
- **删除行** (`GenericDeleted`): 红色 `#FD5B5B`
- **插入行** (`GenericInserted`): 绿色 `#00D787`
- **操作符** (`Operator`): 淡红 `#EF8080`
- **标点** (`Punctuation`): 淡黄 `#E8E8A8`

---

## 使用示例

```go
// 创建自定义样式配置
cfg := ansi.StyleConfig{
    Document: ansi.StyleBlock{
        StylePrimitive: ansi.StylePrimitive{
            Color: new("252"),
        },
        Margin: new(uint(2)),
    },
    Heading: ansi.StyleBlock{
        StylePrimitive: ansi.StylePrimitive{
            Color: new("39"),
            Bold:  new(true),
        },
    },
    // ... 更多配置
}

// 传递给渲染器
renderer := ansi.NewRenderer(ansi.Options{
    StyleConfig: cfg,
})
```

> **辅助函数**: `new` 函数通常是一个简单的指针辅助函数，将值类型转为指针以便区分"未设置"与"零值":
> ```go
> func new[T any](v T) *T { return &v }
> ```

---

## 颜色值格式

`Color` 和 `BackgroundColor` 支持以下格式：

| 格式 | 示例 | 说明 |
|------|------|------|
| ANSI 16色 | `"0"`–`"15"` | 终端基本 16 色 |
| ANSI 256色 | `"0"`–`"255"` | 256 色调色板 |
| 十六进制 | `"#C4C4C4"`, `"#373737"` | RGB 真彩色（需终端支持） |
| ANSI 样式名 | 通过 Lipgloss/termenv 间接支持 | — |
