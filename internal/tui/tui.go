package tui

import (
	"context"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/command"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
	"github.com/lgzzzz/gocode/internal/tui/history"
	"github.com/lgzzzz/gocode/internal/tui/palette"
	"github.com/lgzzzz/gocode/internal/tui/sessionbrowser"
)

// ---- model ----

type model struct {
	editor  textarea.Model
	output  viewport.Model
	agent   *agent.Agent
	history history.History
	palette *palette.Palette // command palette popup

	width  int
	height int

	running bool
	cancel  context.CancelFunc // cancels the running agent context
	ch      chan progressMsg

	store          *store.Store            // session persistence (nil if unavailable)
	sessionID      string                  // current session UUID
	cwd            string                  // current working directory when session was created
	sessionBrowser *sessionbrowser.Browser // session list browser (use Active() to check)
}

// NewModel creates a new TUI model.
func NewModel(ag *agent.Agent, st *store.Store) tea.Model {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width, height = 80, 24
	}

	ta := textarea.New()
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "insert new line"))
	ta.ShowLineNumbers = false // 隐藏行号
	ta.CharLimit = -1          // 无字符限制
	ta.SetVirtualCursor(false) // 使用真实光标（支持闪烁）
	ta.DynamicHeight = true    // 动态高度（自动根据内容调整）
	ta.MinHeight = 1           // 最小 1 行
	ta.MaxHeight = 7           // 最大 7 行
	styles := ta.Styles()
	styles.Cursor.BlinkSpeed = 500 * time.Millisecond
	ta.SetStyles(styles)
	ta.Focus() // 初始获得焦点

	// Initialize command registry and palette.
	reg := command.NewRegistry()
	reg.Register(&command.NewCommand{})
	reg.Register(&command.SessionsCommand{})

	// Generate a session ID (DB row is created lazily on first message).
	cwd, _ := os.Getwd()
	sessionID := store.NewSessionID()

	m := model{
		editor:         ta,
		output:         viewport.New(),
		agent:          ag,
		width:          width,
		height:         height,
		palette:        palette.New(reg),
		store:          st,
		sessionID:      sessionID,
		cwd:            cwd,
		sessionBrowser: sessionbrowser.New(width, height, st),
	}
	m.adjustLayout()
	return m
}

func (m model) Init() tea.Cmd {
	return m.editor.Focus()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		cmds = append(cmds, m.updateOutput(msg)...)
	case tea.PasteMsg:
		cmds = append(cmds, m.updateEditor(msg)...)
	case tea.KeyPressMsg:
		cmds = append(cmds, m.handleKeyPress(msg)...)
	case tea.WindowSizeMsg:
		cmds = append(cmds, m.handleWindowSizeMsg(msg)...)
	case progressMsg:
		cmds = append(cmds, m.handleProgressMsg(msg)...)
	}

	m.adjustLayout()
	m.renderOutput()

	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion

	if m.width == 0 {
		v.SetContent("Initializing...")
		return v
	}

	// Session browser takes over the entire screen when active.
	if m.sessionBrowser.Active() {
		v.SetContent(m.sessionBrowser.View())
		return v
	}

	var editorArea string
	if m.running {
		editorArea = inputBarDimStyle.Render("⏳ Processing... (Esc to stop)")
	} else {
		if m.palette.Active() {
			editorArea = lipgloss.JoinVertical(lipgloss.Left,
				m.palette.Render(),
				m.editor.View(),
			)
		} else {
			editorArea = m.editor.View()
		}
	}

	v.SetContent(lipgloss.JoinVertical(lipgloss.Left,
		m.output.View(),
		"",
		editorArea,
	))

	if !m.running {
		if c := m.editor.Cursor(); c != nil {
			c.Position.Y += m.output.Height() + 1
			if m.palette.Active() {
				c.Position.Y += m.palette.Height()
			}
			v.Cursor = c
		}
	}

	return v
}

// ---- keyboard handling ----

// handleKeyPress processes keyboard events.
func (m *model) handleKeyPress(msg tea.KeyPressMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// ---- session browser mode ----
	if m.sessionBrowser.Active() {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.ExitSessionBrowser()
			return nil
		case "enter":
			if sel := m.sessionBrowser.Selected(); sel != nil {
				m.LoadSession(sel.ID)
			}
			return nil
		default:
			_, cmd := m.sessionBrowser.Update(msg)
			if cmd != nil {
				return []tea.Cmd{cmd}
			}
			return nil
		}
	}

	k := msg.Key()

	// ---- command mode: intercept special keys ----
	if m.palette.Active() {
		result := m.palette.HandleKey(msg.String())

		if result.Dismiss {
			m.editor.Reset()
			return nil
		}

		if result.Execute != nil {
			args := m.palette.Args(result.Execute.Name())
			m.palette.Dismiss()
			m.editor.Reset()
			return []tea.Cmd{m.executeCommand(result.Execute, args)}
		}

		if result.CompleteText != "" {
			m.editor.SetValue(result.CompleteText)
			m.editor.CursorEnd()
			m.palette.UpdateFilter(m.editor.Value())
			return nil
		}

		// For up/down (already handled inside HandleKey) and unrecognized
		// keys, don't forward to the editor — navigation keys are consumed.
		if msg.String() == "up" || msg.String() == "down" {
			return nil
		}

		// For other keys (letters, backspace, etc.), fall through and let the
		// editor process them normally.
	}

	// Always forward to editor (except for special keys that we handle first).
	switch {
	case k.Code == tea.KeyUp || k.Code == tea.KeyDown:
		cmds = append(cmds, m.updateEditor(msg)...)
	default:
		cmds = append(cmds, m.updateEditor(msg)...)
	}

	// After editor update, refresh command palette state
	m.palette.UpdateFilter(m.editor.Value())

	// Special key bindings (quit, submit, etc.)
	switch msg.String() {
	case "ctrl+c":
		cmds = append(cmds, tea.Quit)
		return cmds

	case "esc":
		if m.running {
			m.cancelAgent()
		}
		return cmds

	case "enter":
		if !m.running {
			input := strings.TrimSpace(m.editor.Value())
			if input == "" {
				return cmds
			}
			m.editor.Reset()
			m.history.Append(compoent.NewUserMessage(input))

			// Persist user message (lazily creates session row on first message).
			if m.store != nil {
				m.store.EnsureSession(m.sessionID, m.agent.Model(), m.cwd)
				m.store.AppendMessage(store.Message{
					SessionID: m.sessionID,
					MsgType:   string(agent.MsgUser),
					Content:   input,
				})
			}

			cmd := m.startAgent(input)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.output.GotoBottom()
		}
	}

	return cmds
}

// ---- command execution ----

// executeCommand runs the given command with the provided arguments,
// appends the result to the chat history.  The caller is responsible
// for dismissing the palette and resetting the editor beforehand.
func (m *model) executeCommand(cmd command.Executor, args string) tea.Cmd {
	env := &command.Env{
		Agent: m.agent,
		Model: m, // *model implements ModelAccess directly
	}

	ctx := context.Background()
	result, err := cmd.Execute(ctx, args, env)
	if err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
	} else if result != nil {
		m.history.Append(compoent.NewSystemMessage(result.Message))
	}
	return nil
}

// ---- layout helpers ----

// adjustLayout recalculates editor and output dimensions based on the
// current terminal size, command palette visibility, and editor height.
func (m *model) adjustLayout() {
	m.editor.SetWidth(m.width - 2)
	m.output.SetWidth(m.width - 2)
	m.palette.SetWidth(m.width - 2)

	paletteHeight := m.palette.Height()
	editorHeight := m.editor.Height()
	totalBottom := editorHeight + paletteHeight + 1 // +1 for spacing
	outputHeight := max(0, m.height-totalBottom)
	m.output.SetHeight(outputHeight)

	if m.sessionBrowser.Active() {
		m.sessionBrowser.SetSize(m.width-2, m.height)
	}
}

// handleWindowSizeMsg updates dimensions on terminal resize.
func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) []tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height
	m.history.MarkDirty() // width changed, need re-render
	// WindowSizeMsg still needs to reach editor and output so they
	// can adjust their own internal sizes.
	return append(m.updateEditor(msg), m.updateOutput(msg)...)
}
