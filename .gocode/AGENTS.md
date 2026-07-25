# GoCode - AI Coding Agent

## Project Overview

**GoCode** is a terminal-based AI coding assistant written in Go. It provides an interactive TUI (Terminal User Interface) that connects to the DeepSeek API to help developers read files, execute shell commands, edit code, and write new files — all through natural language conversation.

- **Language**: Go 1.26
- **Module path**: `github.com/lgzzzz/gocode`
- **LLM Backend**: DeepSeek API (OpenAI-compatible), using `deepseek-v4-pro` model
- **TUI Framework**: [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) (`charm.land/bubbletea/v2`)
- **Styling**: [Lipgloss v2](https://github.com/charmbracelet/lipgloss) (`charm.land/lipgloss/v2`)
- **API Client**: `github.com/sashabaranov/go-openai`

## Directory Structure

```
gocode/
├── cmd/
│   └── gocode.go              # Main entry point: sets up env, store, agent, and TUI
├── internal/
│   ├── agent/
│   │   └── agent.go           # Core AI agent: streaming, tool calling, system prompt, history reconstruction
│   ├── command/
│   │   ├── command.go         # Command registry and Executor interface
│   │   ├── init.go            # "init" command — triggers AGENTS.md generation
│   │   ├── new.go             # "new" command — starts a fresh conversation
│   │   ├── prompt.go          # "prompt" command — displays current system prompt
│   │   ├── rollback.go        # "rollback" command — restores files modified by the last agent interaction
│   │   └── sessions.go        # "sessions" command — opens session browser
│   ├── store/
│   │   └── store.go           # Session persistence: JSONL file storage in ~/.gocode/sessions/
│   ├── tools/
│   │   ├── tools.go           # Tool definitions and executors: read, write, edit, bash/powershell
│   │   └── rollback.go        # RollbackTracker: records file modifications for potential rollback
│   └── tui/
│       ├── tui.go             # Main TUI model: layout, key handling, update loop
│       ├── agent.go           # Agent integration: StartAgent, progress handling, persistence
│       ├── editor.go          # Input textarea updates
│       ├── output.go          # Output viewport rendering
│       ├── session.go         # Session management: New, Load, OpenBrowser, CloseBrowser
│       ├── stream.go          # Stream message types and waitCmd helper
│       ├── styles.go          # Shared lipgloss styles
│       ├── compoent/          # Message component types (user, assistant, thinking, tool, error, system)
│       ├── history/           # In-memory message history with dirty-flag rendering
│       ├── palette/           # Slash-command palette UI
│       └── sessionbrowser/    # Session browser/list for loading past sessions
├── go.mod
├── go.sum
├── .gitignore
└── .gocode/
    └── AGENTS.md              # This file — project documentation for AI assistants
```

## Build and Run

### Prerequisites

- Go 1.26+
- A DeepSeek API key

### Environment

Set the API key before running:

```bash
export DEEPSEEK_API_KEY=sk-...
```

On Windows (PowerShell):

```powershell
$env:DEEPSEEK_API_KEY="sk-..."
```

### Build

```bash
go build -o gocode.exe ./cmd/
```

Or run directly:

```bash
go run ./cmd/
```

### Usage

Launch the binary. The TUI shows a chat interface where you type messages and the AI assistant responds. The assistant has access to tools for reading files, writing files, editing files, and running shell commands.

#### Slash Commands

Type `/` in the input to bring up the command palette:

- `/new` — Start a new conversation (clears context)
- `/sessions` — Browse and resume past sessions
- `/init` — Analyze the project and generate a `AGENTS.md` file at `.gocode/AGENTS.md`
- `/prompt` — Display the current system prompt
- `/rollback` — Restore files modified by the last agent interaction (side-effect reversal)

#### Key Bindings

- `Enter` — Submit input
- `Shift+Tab` — Insert newline in the editor
- `Esc` — Cancel a running agent request
- `Ctrl+C` — Quit the application
- `Up/Down` — Navigate in command palette; `Tab` to autocomplete
- `PgUp/PgDown` — Scroll output

## Coding Standards and Conventions

### Go Conventions

- Follow standard Go naming conventions (camelCase for unexported, PascalCase for exported)
- All internal packages are under `internal/` to prevent external import
- Interfaces are small and purpose-specific (`ToolExecutor`, `Logger`, `TUIAccess`, `Component`)
- Use `interface{}` only where necessary; prefer typed structs
- Export only what is needed across packages

### Package Organization

- **`cmd/`**: Single `main` package — wire up dependencies and start the program
- **`internal/agent/`**: Pure AI logic — no UI dependencies; works with callbacks
- **`internal/tools/`**: Tool implementations with JSON-parameter parsing, platform-aware (Windows vs Unix)
- **`internal/command/`**: Slash-command pattern with a `Registry` for dynamic dispatch
- **`internal/store/`**: Thread-safe session persistence using JSONL (one JSON per line)
- **`internal/tui/`**: Bubble Tea model; sub-packages for components, history, palette, and session browser

### Error Handling

- Tools return descriptive errors with context (e.g., `"read %s: %w"`)
- API calls retry up to 3 times with 2-second base delay (exponential backoff)
- Panics are recovered in the agent goroutine and surfaced as error messages

### Concurrency

- The agent runs in a separate goroutine, communicating via a channel of `progressMsg`
- The store uses a `sync.Mutex` to guard concurrent access
- Context-based cancellation is used to stop agent execution

## Key Module Descriptions

### `cmd/gocode.go` — Entry Point

- Reads `DEEPSEEK_API_KEY` from environment
- Opens the session store (defaults to `~/.gocode/sessions/<cwd-path>/`)
- Creates the agent with model `deepseek-v4-pro` and base URL `https://api.deepseek.com`
- Starts the Bubble Tea TUI program

### `internal/agent` — AI Agent

The core of the application. Manages:

- **System prompt**: Auto-generated from available tools and their guidelines. Also loads `AGENTS.md` from the current working directory (specifically `.gocode/AGENTS.md`) if present.
- **Streaming**: Uses OpenAI-compatible streaming API with `reasoning_effort: "max"`. Handles reasoning content and regular content deltas separately, emitting incremental `MsgThinkingStream`/`MsgAssistantStream` callbacks and final `MsgThinking`/`MsgAssistant` callbacks.
- **Tool calling**: Supports native function calling. After receiving tool calls, executes them locally and feeds results back into the conversation loop.
- **Retry logic**: Retries failed API calls up to 3 times with 2-second base delay. When retrying, emits `MsgError` and `MsgRetryWait` callbacks to inform the user.
- **History reconstruction**: `ReconstructHistory()` rebuilds OpenAI-format messages from persisted session data (a list of `HistoryMessage` structs) so past sessions can be resumed with full context.

### `internal/tools` — Tool System

Four tools available to the AI (platform-dependent shell tool):

| Tool | Description | Platform |
|------|-------------|----------|
| `read` | Read file contents with offset/limit support | All |
| `write` | Create or overwrite files (creates parent dirs) | All |
| `edit` | Replace exact text in files (oldText must be unique) | All |
| `bash` | Execute shell commands via bash/sh | Linux/macOS |
| `powershell` | Execute shell commands via PowerShell | Windows |

Each tool implements the `ToolExecutor` interface (`Name()`, `Execute()`, `SetTracker()`). Tool definitions include `PromptSnippet` and `PromptGuidelines` that feed into the auto-generated system prompt to guide the AI on proper tool usage.

#### RollbackTracker

`RollbackTracker` records file modifications and shell commands during each agent interaction. It tracks:
- **File changes**: Original content before modification, whether the file existed, and deduplicates by path (first write wins)
- **Shell commands**: All executed commands and their output

The `/rollback` slash command restores modified files to their original state and reports shell commands that were executed (which cannot be automatically rolled back).

### `internal/command` — Slash Commands

A registry-based command system. Commands implement the `Executor` interface:

```go
type Executor interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args string, env *Env) (*Result, error)
}
```

The `Env` struct provides access to the TUI (via `TUIAccess` interface) for session management, cancellation, and rollback tracker access. The `Result` struct can return a display `Message` and/or an `AgentInput` string that triggers automatic agent invocation (used by `/init`).

Registered commands: `new`, `sessions`, `init`, `prompt`, `rollback`.

### `internal/store` — Session Persistence

Stores sessions as JSONL files in `~/.gocode/sessions/<sanitized-cwd-path>/`. Each file:

- Line 1: Session metadata (ID, created time, model, CWD, first message)
- Subsequent lines: Messages (type, content, tool call info, error status)

Thread-safe with mutex. Supports listing, reading, and appending. The CWD path is sanitized into a directory name (e.g., `C:\Users\...\myproject` becomes `c-users-...-myproject`).

### `internal/tui` — Terminal UI

Built with Bubble Tea v2:

- **Model** holds editor (textarea), output (viewport), agent reference, history, palette, session browser
- **Layout**: Output area on top (scrollable viewport), optional palette, then input editor at bottom. Editor has a 17-line max height.
- **Key handling** is layered: session browser → palette → editor — each layer can consume or pass through keys
- **Rendering**: Uses a dirty-flag pattern in history; only re-renders when content changes
- **Focus**: Editor has initial focus after launch

#### Sub-packages

- **`compoent/`**: Defines a `Component` interface (`Type()`, `MsgID()`, `Content()`, `Render()`, `SetContent()`) and concrete types: `UserMessage`, `AssistantMessage`, `ThinkingMessage`, `ToolMessage`, `ErrorMessage`, `SystemMessage`. Each has a render cache with dirty-flag that only recalculates when content or width changes. Tool messages show a formatted first line (path/command) and up to 6 lines of result, with green styling for success and red for errors.
- **`history/`**: In-memory list of components with `Append` (add new), `Upsert` (update-or-insert by MsgID for streaming updates), `UpdateToolResult` (set tool result on an existing ToolMessage), and dirty-flag-based `Render` that returns line strings only when dirty.
- **`palette/`**: Slash-command palette that appears when typing `/`. Supports filtering by name prefix, keyboard navigation (up/down), tab completion (auto-fills command name), and enter-to-execute. Renders up to 7 visible rows with highlight on selected item.
- **`sessionbrowser/`**: A list-based modal for browsing past sessions. Uses a custom `list.Item` delegate for rendering session items with timestamps and first-message previews. Supports selection via Enter to load a session.

## Notes and Special Conventions

### AGENTS.md Auto-Loading

The agent automatically looks for `.gocode/AGENTS.md` in the current working directory and appends its content to the system prompt. This is how the AI gets project-specific context. The `/init` command can generate this file by asking the AI to analyze the project.

### Session Storage Location

Sessions are stored under `~/.gocode/sessions/` with the CWD path sanitized into a directory name. For example, working in `C:\Users\LGZ\IdeaProjects\gocode` stores sessions in `~/.gocode/sessions/C-Users-LGZ-IdeaProjects-gocode/`.

### Platform Awareness

- On Windows, the `powershell` tool is registered; on Linux/macOS, `bash` is registered
- PowerShell commands use `;` instead of `&&` for chaining
- PowerShell commands are wrapped with `[Console]::OutputEncoding = [System.Text.Encoding]::UTF8;` to handle Unicode
- The system prompt includes the current OS name

### Streaming Updates

Content is streamed incrementally. The TUI uses `Upsert` (update-or-insert) for streaming messages so the display updates in real-time without duplicating entries. The final non-streaming message is also emitted when streaming completes.

### Tool Call Flow

1. AI requests tool calls in its response (in parallel or sequentially)
2. Agent emits `MsgToolCall` for each call (with a unique ID)
3. Tool is executed locally via `ToolExecutor.Execute()`
4. Result is emitted as `MsgToolResult` (linked by ID), with error flag if the tool failed or returned non-zero exit code
5. Results are fed back to the API for the next turn
6. Loop continues until the AI produces a final text response (no more tool calls)

### Rollback Flow

1. Each agent interaction creates a fresh `RollbackTracker`
2. `write` and `edit` tools record original file state before modifying
3. Shell tools record the command and output
4. The user can invoke `/rollback` to restore files and review shell commands
5. After rollback or starting a new agent interaction, the tracker is cleared
