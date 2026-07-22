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
)

// ---- model ----

type model struct {
	editor  textarea.Model
	output  viewport.Model
	agent   *agent.Agent
	history History
	width   int
	height  int
	running bool
	cancel  context.CancelFunc // cancels the running agent context
	ch      chan progressMsg

	lastContent string // track content to detect when it changes (for auto-scroll)
}

// NewModel creates a new TUI model.
func NewModel(ag *agent.Agent) tea.Model {
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

	// 设置光标闪烁速度（默认 530ms，这里调快一些）
	styles := ta.Styles()
	styles.Cursor.BlinkSpeed = 500 * time.Millisecond
	ta.SetStyles(styles)

	ta.Focus() // 初始获得焦点

	m := model{
		editor: ta,
		output: viewport.New(),
		agent:  ag,
		width:  width,
		height: height,
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

	var editorArea string
	if m.running {
		editorArea = inputBarDimStyle.Render("⏳ Processing... (Esc to stop)")
	} else {
		editorArea = m.editor.View()
	}

	// Place the editor below the output area.
	outputContent := m.output.View()
	v.SetContent(lipgloss.JoinVertical(lipgloss.Left,
		outputContent,
		"",
		editorArea,
	))

	// Set real cursor position for the editor.
	// The editor cursor is relative to its own top-left; offset it by the
	// output height + 1 blank line to match its position in the full view.
	if !m.running {
		if c := m.editor.Cursor(); c != nil {
			c.Position.Y += m.output.Height() + 1
			v.Cursor = c
		}
	}

	return v
}

func (m *model) adjustLayout() {
	m.editor.SetWidth(m.width - 2)
	m.output.SetWidth(m.width - 2)
	m.output.SetHeight(max(0, m.height-m.editor.Height()-1))
}
