# GoCode 更新计划

## 1. 底部状态栏

在主界面底部增加状态栏，实时展示当前工作上下文信息：

- **当前工作目录** — 显示当前项目的绝对路径或相对于 `$HOME` 的简洁路径。
- **Git 分支** — 自动检测当前目录所在的 Git 仓库，展示分支名（如 `main`、`feature/xxx`）。非 Git 目录则显示 `N/A`。
- **Token 使用情况** — 统计并展示当前会话已消耗的 prompt / completion token 数量（基于 API 返回的 `usage` 字段累加）。
- **上下文情况** — 显示当前会话上下文中的消息数量及大致占比（已用 / 最大上下文窗口）。

---

## 2. Skill 支持

引入 **Skill** 机制，允许用户为 Agent 定义可复用的能力模块。每个 Skill 是一组预定义的：

- **系统提示词片段** — 描述该 Skill 的职责、行为准则、领域知识。
- **工具集** — 该 Skill 专属或共享的工具（可选）。
- **触发条件** — 用户可通过 `/skill <name>` 激活，或在对话中由 Agent 自动匹配。

### 使用场景举例

| Skill | 描述 |
|---|---|
| `code-review` | 对指定文件或改动进行代码审查，输出结构化 Review 意见 |
| `refactor` | 按照 SOLID 原则重构选定代码 |
| `doc-gen` | 为指定函数/包生成 API 文档 |
| `test-gen` | 为指定函数/包生成单元测试 |

Skill 将以配置文件（如 `.gocode/skills/` 目录下的 YAML/JSON 文件）形式定义，支持用户自行创建和分享。

---

## 3. 项目 SDK 化

将当前 GoCode 重构为 **可嵌入的 SDK**，核心思路：

- 将 `internal/` 下的核心模块（`agent`、`tools`、`tui`、`store`、`command`）提炼为可导出的公开包（`pkg/`）。
- `cmd/gocode.go` 变为 SDK 的一个**默认参考实现**，用户只需极少的代码即可启动自己的定制版 GoCode：

```go
package main

import (
    "github.com/lgzzzz/gocode/pkg/agent"
    "github.com/lgzzzz/gocode/pkg/tui"
    "github.com/lgzzzz/gocode/pkg/tools"
)

func main() {
    cfg := agent.DefaultConfig()
    cfg.APIKey = os.Getenv("MY_API_KEY")
    cfg.Model  = "custom-model"

    app := tui.New(cfg)
    app.Use(tools.Read, tools.Write, tools.Edit, tools.Shell)
    app.Run()
}
```

用户只需 `go build` 即可得到一个完全可用的 GoCode，且可以自由替换模型、工具、界面等。

---

## 4. SDK 扩展体系（中间件模式）

在 SDK 化基础之上，提供一套 **事件驱动的中间件（Middleware）扩展体系**。

### 核心概念

GoCode 在运行过程中会产生一系列**生命周期事件**：

```
UserInput → PreProcess → AgentCall → StreamResponse → ToolCall → ToolResult → PostProcess → Render
```

用户可以注册**扩展（Extension）**，以中间件的形式插入上述事件流，依次对事件进行处理或拦截。

### API 设计（草案）

```go
// Extension 定义了扩展接口
type Extension interface {
    Name() string
    // Hook 返回该扩展关心的事件类型列表
    Hook() []EventType
    // Handle 处理事件，返回处理后的事件和可选的错误
    Handle(ctx context.Context, event Event) (Event, error)
}
```

### 典型扩展场景

| 扩展 | 功能 |
|---|---|
| `Logger` | 记录所有请求/响应到日志文件 |
| `RateLimiter` | 限制 API 调用频率 |
| `Cache` | 缓存 LLM 响应，减少重复调用 |
| `Validator` | 校验工具调用的参数合法性 |
| `Notifier` | 任务完成时发送桌面通知 |
| `CustomTool` | 注册用户自定义工具（如数据库查询、API 调用） |

### 注册方式

```go
app.Use(myExtension)
app.UseTool(myCustomTool)
app.OverrideTool("edit", myEnhancedEditTool)  // 覆盖默认工具
```

### 中间件执行顺序

多个扩展按注册顺序依次执行，每个扩展可以选择：

- **继续传递**（返回修改后的事件）
- **短路拦截**（不调用后续扩展，直接返回）
- **注入副作用**（如日志、监控）

---

## 时间线（暂定）

| 阶段 | 内容 | 预期 |
|---|---|---|
| **Phase 1** | 底部状态栏 | 近期 |
| **Phase 2** | Skill 支持 | 中期 |
| **Phase 3** | 项目 SDK 化重构 | 中长期 |
| **Phase 4** | SDK 扩展 & 中间件体系 | 长期 |

> 以上计划可能会根据实际情况调整，欢迎通过 Issue 提出建议和反馈。
