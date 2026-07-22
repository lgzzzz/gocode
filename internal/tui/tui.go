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
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- model ----

type model struct {
	input    textarea.Model
	viewport viewport.Model
	agent    *agent.Agent
	log      []compoent.Component
	width    int
	height   int
	running  bool
	cancel   context.CancelFunc // cancels the running agent context
	ch       chan progressMsg

	lastContent string // track content to detect when it changes (for auto-scroll)
	dirty       bool   // true when log needs re-rendering
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
		input:    ta,
		viewport: viewport.New(),
		agent:    ag,
		width:    width,
		height:   height,
	}
	m.adjustLayout()
	return m
}

func (m model) Init() tea.Cmd {
	return m.input.Focus()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		cmds = append(cmds, m.updateViewportModel(msg)...)
	case tea.PasteMsg:
		cmds = append(cmds, m.updateInput(msg)...)
	case tea.KeyPressMsg:
		cmds = append(cmds, m.handleKeyPress(msg)...)
	case tea.WindowSizeMsg:
		cmds = append(cmds, m.handleWindowSizeMsg(msg)...)
	case progressMsg:
		cmds = append(cmds, m.handleProgressMsg(msg)...)
	}

	m.adjustLayout()
	m.updateViewport()

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

	var inputArea string
	if m.running {
		inputArea = inputBarDimStyle.Render("⏳ Processing... (Esc to stop)")
	} else {
		inputArea = m.input.View()
	}

	// Put input area inside the viewport at the bottom
	vpContent := m.viewport.View()
	v.SetContent(lipgloss.JoinVertical(lipgloss.Left,
		vpContent,
		"",
		inputArea,
	))

	// Set real cursor position for the textarea.
	// The textarea cursor is relative to its own top-left; offset it by the
	// viewport height + 1 blank line to match its position in the full view.
	if !m.running {
		if c := m.input.Cursor(); c != nil {
			c.Position.Y += m.viewport.Height() + 1
			v.Cursor = c
		}
	}

	return v
}

// appendLog adds a component to the log and marks it dirty for re-render.
func (m *model) appendLog(c compoent.Component) {
	m.log = append(m.log, c)
	m.dirty = true
}

func (m *model) adjustLayout() {
	m.input.SetWidth(m.width - 2)
	m.viewport.SetWidth(m.width - 2)
	m.viewport.SetHeight(max(0, m.height-m.input.Height()-1))
}
