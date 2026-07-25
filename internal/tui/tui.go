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
	"github.com/lgzzzz/gocode/internal/tools"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
	"github.com/lgzzzz/gocode/internal/tui/history"
	"github.com/lgzzzz/gocode/internal/tui/palette"
	"github.com/lgzzzz/gocode/internal/tui/sessionbrowser"
)

type model struct {
	editor  textarea.Model
	output  viewport.Model
	agent   *agent.Agent
	history history.History
	palette *palette.Palette

	width  int
	height int

	running bool
	cancel  context.CancelFunc
	ch      chan progressMsg

	store          *store.Store
	sessionID      string
	cwd            string
	sessionBrowser *sessionbrowser.Browser
	rollbackTracker *tools.RollbackTracker
}

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
	styles := ta.Styles()
	styles.Cursor.BlinkSpeed = 500 * time.Millisecond
	ta.SetStyles(styles)
	ta.Focus() // 初始获得焦点

	rollbackTracker := tools.NewRollbackTracker()
	ag.SetToolTracker(rollbackTracker)

	reg := command.NewRegistry()
	reg.Register(&command.NewCommand{})
	reg.Register(&command.SessionsCommand{})
	reg.Register(&command.InitCommand{})
	reg.Register(&command.PromptCommand{})
	reg.Register(&command.RollbackCommand{})

	cwd, _ := os.Getwd()
	sessionID := store.NewSessionID()

	m := model{
		editor:          ta,
		output:          viewport.New(),
		agent:           ag,
		width:           width,
		height:          height,
		palette:         palette.New(reg),
		store:           st,
		sessionID:       sessionID,
		cwd:             cwd,
		sessionBrowser:  sessionbrowser.New(width, height, st),
		rollbackTracker: rollbackTracker,
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

// handleKeyPress dispatches key events to the appropriate handler based on UI mode.
// Layer 1: Session browser modal → handleSessionBrowserKey
// Layer 2: Command palette → handlePaletteKey (unconsumed keys pass through)
// Layer 3: Default editing mode → handleEditingKey
func (m *model) handleKeyPress(msg tea.KeyPressMsg) []tea.Cmd {
	if m.sessionBrowser.Active() {
		return m.handleSessionBrowserKey(msg)
	}

	if m.palette.Active() {
		consumed, cmds := m.handlePaletteKey(msg)
		if consumed {
			return cmds
		}
	}

	k := msg.Key()
	if k.Code == tea.KeyPgUp || k.Code == tea.KeyPgDown {
		return m.updateOutput(msg)
	}

	return m.handleEditingKey(msg)
}

// handleSessionBrowserKey handles keys when the session browser modal is active.
func (m *model) handleSessionBrowserKey(msg tea.KeyPressMsg) []tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.CloseSessionBrowser()
	case "enter":
		if sel := m.sessionBrowser.Selected(); sel != nil {
			m.LoadSession(sel.SessionID)
		}
	default:
		_, cmd := m.sessionBrowser.Update(msg)
		if cmd != nil {
			return []tea.Cmd{cmd}
		}
	}
	return nil
}

// handlePaletteKey handles keys when the command palette is active.
// Returns consumed=true if the palette fully handled the key;
// otherwise the key passes through to the editor.
func (m *model) handlePaletteKey(msg tea.KeyPressMsg) (consumed bool, cmds []tea.Cmd) {
	result := m.palette.HandleKey(msg.String())

	switch {
	case result.Dismiss:
		m.editor.Reset()
		return true, nil
	case result.Execute != nil:
		args := m.palette.Args(result.Execute.Name())
		m.palette.Dismiss()
		m.editor.Reset()
		return true, []tea.Cmd{m.executeCommand(result.Execute, args)}
	case result.CompleteText != "":
		m.editor.SetValue(result.CompleteText)
		m.editor.CursorEnd()
		m.palette.UpdateFilter(m.editor.Value())
		return true, nil
	}

	// Palette consumes arrow keys for navigation
	if msg.String() == "up" || msg.String() == "down" {
		return true, nil
	}

	return false, nil
}

// handleEditingKey handles keys in normal editing mode:
// editor input, output scrolling, global shortcuts, and input submission.
func (m *model) handleEditingKey(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return append(cmds, tea.Quit)
	case "esc":
		if m.running {
			m.cancelAgent()
		}
		return cmds
	case "enter":
		if cmd := m.submitInput(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return cmds
	}

	cmds = m.updateEditor(msg)

	m.palette.UpdateFilter(m.editor.Value())

	return cmds
}

// submitInput validates, persists, and submits the user's input to the agent.
func (m *model) submitInput() tea.Cmd {
	if m.running {
		return nil
	}
	input := strings.TrimSpace(m.editor.Value())
	if input == "" {
		return nil
	}

	m.editor.Reset()
	m.history.Append(compoent.NewUserMessage(input))
	m.persistUserInput(input)
	m.output.GotoBottom()

	return m.StartAgent(input)
}

// persistUserInput saves the user's input to the session store.
func (m *model) persistUserInput(input string) {
	if m.store == nil {
		return
	}
	m.store.EnsureSession(m.sessionID, m.agent.Model(), m.cwd)
	m.store.AppendMessage(store.Message{
		SessionID: m.sessionID,
		MsgType:   string(agent.MsgUser),
		Content:   input,
	})
}

func (m *model) executeCommand(cmd command.Executor, args string) tea.Cmd {
	env := &command.Env{
		TUI: m,
	}

	ctx := context.Background()
	result, err := cmd.Execute(ctx, args, env)
	if err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
		return nil
	}
	if result == nil {
		return nil
	}
	if result.Message != "" {
		m.history.Append(compoent.NewSystemMessage(result.Message))
	}
	if result.AgentInput != "" {
		m.history.Append(compoent.NewUserMessage(result.AgentInput))
		m.persistUserInput(result.AgentInput)
		m.output.GotoBottom()
		return m.StartAgent(result.AgentInput)
	}
	return nil
}

func (m *model) adjustLayout() {
	m.editor.SetWidth(m.width - 2)
	m.output.SetWidth(m.width - 2)
	m.palette.SetWidth(m.width - 2)

	paletteHeight := m.palette.Height()
	editorHeight := m.editor.Height()
	if editorHeight > 17 {
		editorHeight = 17
		m.editor.SetHeight(editorHeight)
	}
	totalBottom := editorHeight + paletteHeight + 1
	outputHeight := max(0, m.height-totalBottom)
	m.output.SetHeight(outputHeight)

	if m.sessionBrowser.Active() {
		m.sessionBrowser.SetSize(m.width-2, m.height)
	}
}

func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) []tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height
	m.history.MarkDirty()
	return append(m.updateEditor(msg), m.updateOutput(msg)...)
}
