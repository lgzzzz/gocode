# GoCode — Project Overview

**GoCode** is an interactive terminal-based AI coding assistant (TUI) written in Go. It connects to the **DeepSeek API** (OpenAI-compatible) to provide a conversational agent that can read, write, and edit files, execute shell commands, and maintain context across sessions — all from within your terminal.

- **Language:** Go 1.26
- **Module:** `github.com/lgzzzz/gocode`
- **TUI framework:** Bubble Tea v2 + Bubbles v2 + Lipgloss v2 + Glamour v2 (Markdown rendering)
- **AI backend:** DeepSeek API via `go-openai`
- **Platform:** Windows (primary), with Linux/macOS cross-platform support

---

## Directory Structure

```
gocode/
├── cmd/
│   └── gocode.go              # Application entry point
├── internal/
│   ├── agent/
│   │   └── agent.go           # LLM agent: streaming, system prompt, context, tool orchestration
│   ├── command/
│   │   ├── command.go         # Command executor interface & registry
│   │   ├── init.go            # /init — generate AGENTS.md
│   │   ├── new.go             # /new  — start a new conversation
│   │   ├── prompt.go          # /prompt — display the current system prompt
│   │   ├── rollback.go        # /rollback — revert file changes from the last interaction
│   │   ├── sessions.go        # /sessions — browse and continue previous sessions
│   │   └── update.go          # /update — update AGENTS.md from recent project changes
│   ├── store/
│   │   └── store.go           # Session & message persistence (JSONL file store)
│   ├── tools/
│   │   ├── tools.go           # Tool definitions: read, write, edit, bash/powershell
│   │   └── rollback.go        # Rollback tracker: records file/shell side effects for undo
│   └── tui/
│       ├── tui.go             # Main TUI model (Bubble Tea)
│       ├── agent.go           # Agent ↔ TUI bridge (goroutine + channel)
│       ├── editor.go          # Text input area wrapper
│       ├── output.go          # Scrollable output viewport
│       ├── session.go         # Session lifecycle: new, load, browser
│       ├── stream.go          # Streaming progress messages from agent
│       ├── styles.go          # Global TUI styles (input bar dim style)
│       ├── compoent/          # TUI component types (renderable message blocks)
│       │   ├── component.go   # Component interface
│       │   ├── assistant.go   # Assistant message block (Markdown rendered)
│       │   ├── user.go        # User message block
│       │   ├── thinking.go    # LLM reasoning/thinking block (Markdown rendered)
│       │   ├── tool.go        # Tool call / result block
│       │   ├── error.go       # Error message block
│       │   ├── system.go      # System message block
│       │   ├── styles.go      # Component-specific lipgloss styles
│       │   └── thinking_test.go  # Thinking component unit test
│       ├── history/
│       │   └── history.go     # Ordered message list with upsert & truncation
│       ├── markdown/
│       │   ├── markdown.go    # Streaming-optimized Markdown renderer (glamour)
│       │   └── splitdetector.go  # Safe split-point detection for incremental rendering
│       ├── palette/
│       │   └── palette.go     # Slash-command palette UI
│       ├── sessionbrowser/
│       │   └── sessionbrowser.go  # Session picker modal
│       ├── test/
│       │   ├── markdown_test.go   # Markdown rendering performance tests
│       │   └── testMarkdown.md    # Test fixture for markdown tests
│       └── util/
│           └── string.go      # ANSI-aware string utilities (trim, empty checks)
├── .gocode/
│   └── AGENTS.md              # This file (auto-loaded as additional system prompt context)
├── go.mod
├── go.sum
└── .gitignore
```

---

## Build & Run

### Prerequisites

- Go 1.26 or later
- A DeepSeek API key set as the environment variable `DEEPSEEK_API_KEY`

### Run (development)

```bash
export DEEPSEEK_API_KEY=sk-...
go run ./cmd/gocode.go
```

### Build

```bash
go build -o gocode ./cmd/gocode.go
```

### Test

```bash
go test ./internal/...
```

Test files exist for:
- **Markdown rendering** (`internal/tui/test/markdown_test.go`): Performance benchmarks comparing full, glamour-cached, and streaming-cached rendering modes using a multi-thousand-word test fixture.
- **Thinking component** (`internal/tui/compoent/thinking_test.go`): Basic rendering smoke test for the thinking message component.

Manual verification is also done by running the TUI and interacting with the agent.

---

## Coding Standards & Conventions

### General
- **Language:** All identifiers, comments, and documentation use **English**.
- **Package naming:** Lowercase, single-word where possible (`agent`, `command`, `tools`, `store`, `tui`).
- **File naming:** Lowercase with underscores only when necessary (`sessionbrowser.go`).
- **Error handling:** Errors are returned as values and handled explicitly. The agent reports errors via `CallbackMsg` with type `MsgError`; tools return errors as formatted strings.

### Naming Conventions
- **Exported types/functions:** PascalCase (`NewModel`, `CallbackMsg`, `RollbackTracker`).
- **Unexported types/functions:** camelCase (`sysMsg`, `userMsg`, `loadAgentsMD`, `render`).
- **Interfaces:** Suffix `-er` when describing a capability (`ToolExecutor`, `TUIAccess`, `Executor`).

### Code Organization
- **`internal/`** contains all application logic; nothing outside `internal/` is importable by other modules.
- **TUI components** implement the `Component` interface (from `internal/tui/compoent/component.go`) with `Type()`, `MsgID()`, `Content()`, `Render()`, and `SetContent()`.
- **Tools** implement `ToolExecutor` (from `internal/tools/tools.go`) with `Name()`, `Execute()`, and `SetTracker()`.
- **Commands** implement `command.Executor` with `Name()`, `Description()`, and `Execute()`.

### Style
- Component lipgloss styles are defined in a single `var` block in `internal/tui/compoent/styles.go` using the `leftBar()` helper. Global TUI styles (e.g., input bar dim style) are in `internal/tui/styles.go`.
- Message types are string constants defined in `agent.MsgType`.
- Rendering uses a lazy caching pattern: components cache rendered output and only re-render when `dirty` is true or the width changes. Assistant and Thinking components additionally use the streaming-optimized `markdown.Renderer`.

---

## Key Modules

### `cmd/gocode.go` — Entry Point
Reads `DEEPSEEK_API_KEY` from the environment, initializes the session store, creates the agent with the DeepSeek model (`deepseek-v4-pro`), and launches the Bubble Tea TUI program.

### `internal/agent` — LLM Agent
Core agent that communicates with the DeepSeek API. Key responsibilities:
- **System prompt construction:** Dynamically builds a system prompt that includes available tool descriptions, usage guidelines, the current working directory, OS information, and optionally the contents of `.gocode/AGENTS.md`. The `skipAgentsMD` flag suppresses AGENTS.md loading for one invocation (used by `/init` and `/update`).
- **Streaming:** Uses OpenAI-compatible streaming API with `ReasoningEffort: "max"`; emits incremental `CallbackMsg` events for real-time UI updates (thinking stream, assistant stream, tool calls, tool results).
- **Context management:** Maintains conversation history (`contextMessages`), supports truncation from the last user message (`TruncateContextFromLastUser`), full context replacement (`SetContextMessage`), and reconstruction from stored history (`ReconstructHistory`).
- **Retry logic:** Automatically retries failed API calls up to 3 times with a 2-second backoff.
- **Tool orchestration:** After receiving tool calls from the LLM, executes them sequentially and feeds results back into the conversation loop. Tools receive a shared `RollbackTracker` via `SetToolTracker`.

### `internal/tools` — Tool System
Provides the tools the agent can use:

| Tool        | Description                                               | Side Effects Tracked? |
|-------------|-----------------------------------------------------------|-----------------------|
| `read`      | Read file contents (with offset/limit)                    | No                    |
| `write`     | Create or overwrite a file                                | Yes (file content)    |
| `edit`      | Replace a unique text occurrence in a file (normalizes line endings to LF) | Yes (file content) |
| `powershell` (Windows) / `bash` (Linux/macOS) | Execute shell commands (30s default timeout) | Yes (command log) |

- **`RollbackTracker`:** Records original file contents before modification (first write per path wins) and shell commands executed. The `/rollback` command restores files to their pre-modification state, reports shell commands that may have side effects, and rolls back the conversation context to before the last interaction.

### `internal/tui` — Terminal User Interface
Built with Bubble Tea v2 + Bubbles v2. The TUI consists of:
- **Output area** (viewport): Scrollable message history with distinct styles for user, assistant, thinking, tool calls/results, errors, and system messages.
- **Input area** (textarea): Multi-line text input with dynamic height (max 17 lines).
- **Command palette:** Activated by typing `/`; provides autocomplete for slash commands (`/new`, `/init`, `/update`, `/sessions`, `/prompt`, `/rollback`).
- **Session browser:** Modal list for browsing and resuming previous sessions.
- **Markdown rendering:** Uses Glamour v2 for rich Markdown output. The `markdown` package provides a streaming-optimized cached renderer that incrementally re-renders only the changed tail of streaming content, using safe split-point detection (blank lines, unclosed fences).
- **ANSI utilities:** The `util` package provides helpers for trimming invisible/empty lines from ANSI-styled strings.

**Message flow:** User types input → Enter → Agent goroutine streams responses via channel → TUI receives `progressMsg` → History updates → Viewport re-renders (ticked at 25ms intervals during agent processing).

### `internal/command` — Slash Commands
Commands are registered in a `Registry` and invoked via the command palette or by typing `/command`:

| Command      | Description                                                  |
|--------------|--------------------------------------------------------------|
| `/new`       | Start a fresh conversation (clears context)                  |
| `/init`      | Analyze the project and generate `.gocode/AGENTS.md`         |
| `/update`    | Analyze recent project changes and update `.gocode/AGENTS.md` |
| `/sessions`  | Open the session browser to continue a previous session      |
| `/prompt`    | Display the current system prompt (for debugging)            |
| `/rollback`  | Revert file changes and conversation context from the last agent interaction |

### `internal/store` — Persistence
Stores sessions and messages as JSONL files under `~/.gocode/sessions/<cwd-hash>/`. Each session is a `.session` file with the first line as session metadata and subsequent lines as individual messages. Supports listing, loading, appending, and truncation.

---

## Notes & Special Conventions

1. **AGENTS.md auto-loading:** When the agent starts a conversation, it looks for `.gocode/AGENTS.md` in the current working directory. If found, its content is appended to the system prompt, giving the AI context about the project. The `/init` command generates this file; the `/update` command refreshes it based on recent project changes. Both commands set `skipAgentsMD` to avoid recursively loading the file during generation.

2. **API key:** The application requires `DEEPSEEK_API_KEY` to be set. It will exit with an error if the variable is missing.

3. **Platform-aware shell tool:** On Windows the agent gets a `powershell` tool (with UTF-8 output encoding and `;` command separator); on Linux/macOS it gets a `bash` tool (with `&&` command separator).

4. **Rollback scope:** The `/rollback` command performs three actions: (a) restores files modified during the last interaction to their original state; (b) reports shell commands that were executed (which must be manually reversed); and (c) truncates the conversation context (agent, TUI history, and persisted store) to before the last user message.

5. **Session files** are stored at `~/.gocode/sessions/<sanitized-cwd-path>/` using a sanitized version of the project's absolute path, ensuring different projects have separate session histories.

6. **Streaming IDs:** Streaming messages (thinking and assistant) use UUIDs to identify update targets, allowing the TUI to upsert rather than append duplicate entries during incremental streaming. Both the final thinking and assistant messages are emitted after their respective streams complete.

7. **No external configuration file:** All configuration is via environment variables or defaults. The model (`deepseek-v4-pro`) and base URL (`https://api.deepseek.com`) are hardcoded in `cmd/gocode.go`.

8. **Markdown rendering:** Assistant and Thinking messages are rendered through Glamour (with the Dracula theme, compact margins). The renderer caches a "stable prefix" and only re-renders the tail during streaming, using blank-line boundaries to find safe split points while respecting unclosed fenced code blocks.

9. **Tool result display:** Tool results in the TUI are truncated to 6 lines with an "...N more lines..." indicator for long outputs.

10. **Edit tool line endings:** The edit tool normalizes all line endings to LF (`\n`) before performing replacements, converting both CRLF and CR to LF.
