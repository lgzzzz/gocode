# GoCode

Interactive terminal-based AI coding assistant.

> [中文版本](README.md) | [Roadmap](ROADMAP.md)

---

## Prerequisites

- **Go 1.26+**
- **DeepSeek API Key** ([get one](https://platform.deepseek.com/))

---

## Installation

```bash
git clone https://github.com/lgzzzz/gocode.git
cd gocode
go build -o gocode ./cmd/gocode.go
```

---

## Running

```bash
# Set API Key
# Linux/macOS
export DEEPSEEK_API_KEY=sk-your-api-key-here

# Windows (PowerShell)
$env:DEEPSEEK_API_KEY = "sk-your-api-key-here"

# Launch
gocode
```

---

## Usage

### Chat

Type your request and press **Enter** to send. The agent can read files, write code, edit files, and execute shell commands.

Use **Shift+Tab** for new lines.

### Slash Commands

Type `/` to open the command palette, or enter a command directly:

| Command | Description |
|---|---|
| `/new` | Start a new conversation (clears context) |
| `/init` | Analyze the project and generate `.gocode/AGENTS.md` |
| `/update` | Update `.gocode/AGENTS.md` |
| `/sessions` | Browse and continue previous sessions |
| `/prompt` | Show the current system prompt |
| `/rollback` | Revert file changes from the last interaction |

### Keybindings

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Shift+Tab` | New line |
| `/` | Open command palette |
| `Esc` | Close palette / close session browser |
| `Ctrl+C` | Quit |
