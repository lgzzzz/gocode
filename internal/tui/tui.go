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

	reg := command.NewRegistry()
	reg.Register(&command.NewCommand{})
	reg.Register(&command.SessionsCommand{})

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


func (m *model) handleKeyPress(msg tea.KeyPressMsg) []tea.Cmd {
	var cmds []tea.Cmd

	if m.sessionBrowser.Active() {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.CloseSessionBrowser()
			return nil
		case "enter":
			if sel := m.sessionBrowser.Selected(); sel != nil {
				m.LoadSession(sel.SessionID)
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

		if msg.String() == "up" || msg.String() == "down" {
			return nil
		}

	}

	if k.Code == tea.KeyPgUp || k.Code == tea.KeyPgDown {
		cmds = append(cmds, m.updateOutput(msg)...)
		return cmds
	}

	switch {
	case k.Code == tea.KeyUp || k.Code == tea.KeyDown:
		cmds = append(cmds, m.updateEditor(msg)...)
	default:
		cmds = append(cmds, m.updateEditor(msg)...)
	}

	m.palette.UpdateFilter(m.editor.Value())

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

			if m.store != nil {
				m.store.EnsureSession(m.sessionID, m.agent.Model(), m.cwd)
				m.store.AppendMessage(store.Message{
					SessionID: m.sessionID,
					MsgType:   string(agent.MsgUser),
					Content:   input,
				})
			}

			cmd := m.StartAgent(input)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.output.GotoBottom()
		}
	}

	return cmds
}


func (m *model) executeCommand(cmd command.Executor, args string) tea.Cmd {
	env := &command.Env{
		TUI: m,
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
