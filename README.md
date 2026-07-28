# GoCode

交互式终端 AI 编码助手。

> [English Version](README.en.md)

---

## 环境要求

- **Go 1.26+**
- **DeepSeek API Key**（[获取](https://platform.deepseek.com/)）

---

## 安装

```bash
git clone https://github.com/lgzzzz/gocode.git
cd gocode
go build -o gocode ./cmd/gocode.go
```

---

## 运行

```bash
# 设置 API Key
# Linux/macOS
export DEEPSEEK_API_KEY=sk-your-api-key-here

# Windows (PowerShell)
$env:DEEPSEEK_API_KEY = "sk-your-api-key-here"

# 启动
gocode
```

---

## 使用方法

### 对话

直接输入问题或需求，按 **Enter** 发送。Agent 可以读取文件、编写代码、编辑文件、执行 Shell 命令。

换行使用 **Shift+Tab**。

### Slash 命令

输入 `/` 打开命令面板，或直接输入命令：

| 命令 | 说明 |
|---|---|
| `/new` | 开始新对话（清空上下文） |
| `/init` | 分析项目并生成 `.gocode/AGENTS.md` |
| `/update` | 更新 `.gocode/AGENTS.md` |
| `/sessions` | 浏览并继续历史会话 |
| `/prompt` | 查看当前系统提示词 |
| `/rollback` | 撤销上次交互中 Agent 对文件的修改 |

### 快捷键

| 按键 | 功能 |
|---|---|
| `Enter` | 发送消息 |
| `Shift+Tab` | 换行 |
| `/` | 打开命令面板 |
| `Esc` | 关闭面板 / 关闭会话浏览器 |
| `Ctrl+C` | 退出 |
