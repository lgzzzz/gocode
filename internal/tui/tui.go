package tui

import (
	"context"
	"os"
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
