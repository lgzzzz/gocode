package tui

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- progress message ----

type progressMsg struct {
	typ      agent.MsgType // callback message type
	id       string        // message ID (for streaming updates)
	content  string
	toolName string // tool name (set for tool_call)
	toolArgs string // tool arguments JSON (set for tool_call)
	toolErr  error  // tool execution error (set for tool_result)
	done     bool
	err      error // fatal / panic error
}

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
	case tea.KeyPressMsg:
		cmds = append(cmds, m.handleKeyPress(msg)...)
	case tea.KeyReleaseMsg:
		// ignored for now
	case tea.WindowSizeMsg:
		cmds = append(cmds, m.handleWindowSizeMsg(msg)...)
	case progressMsg:
		cmds = append(cmds, m.handleProgressMsg(msg)...)
	}

	m.adjustLayout()
	m.updateViewport()

	return m, tea.Batch(cmds...)
}

// ---- message handlers ----

// handleKeyPress processes keyboard events.
func (m *model) handleKeyPress(msg tea.KeyPressMsg) []tea.Cmd {
	var cmds []tea.Cmd

	k := msg.Key()

	// Always forward to input (except for special keys that we handle first).
	switch {
	case k.Code == tea.KeyUp || k.Code == tea.KeyDown:
		cmds = append(cmds, m.updateInput(msg)...)
	default:
		cmds = append(cmds, m.updateInput(msg)...)
	}

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
			cmd := m.submitTask()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return cmds
}

// handleWindowSizeMsg updates dimensions on terminal resize.
func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) []tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height
	m.dirty = true // width changed, need re-render
	// WindowSizeMsg still needs to reach input and viewport so they
	// can adjust their own internal sizes.
	return append(m.updateInput(msg), m.updateViewportModel(msg)...)
}

// handleProgressMsg processes agent callback messages (streaming, tool calls, etc.).
func (m *model) handleProgressMsg(msg progressMsg) []tea.Cmd {
	if msg.err != nil {
		m.log = append(m.log,
			compoent.ErrorMessage{Content: msg.err.Error()},
		)
		m.dirty = true
		return nil
	}

	if msg.done {
		m.running = false
		m.cancel = nil
		m.ch = nil
		return nil
	}

	switch msg.typ {
	case agent.MsgAssistantStream, agent.MsgThinkingStream:
		m.applyStreamUpdate(msg)

	case agent.MsgToolCall:
		m.log = append(m.log, compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs))
		m.dirty = true

	case agent.MsgToolResult:
		m.applyToolResult(msg)
		m.dirty = true

	default:
		m.log = append(m.log, &compoent.AssistantMessage{ID: msg.id, Content: msg.content})
		m.dirty = true
	}

	if m.ch != nil {
		return []tea.Cmd{waitCmd(m.ch)}
	}
	return nil
}

// ---- sub-component helpers ----

// updateInput forwards a message to the input textarea and returns any command.
func (m *model) updateInput(msg tea.Msg) []tea.Cmd {
	newInput, cmd := m.input.Update(msg)
	m.input = newInput
	if cmd != nil {
		return []tea.Cmd{cmd}
	}
	return nil
}

// updateViewportModel forwards a message to the viewport and returns any command.
func (m *model) updateViewportModel(msg tea.Msg) []tea.Cmd {
	newVP, cmd := m.viewport.Update(msg)
	m.viewport = newVP
	if cmd != nil {
		return []tea.Cmd{cmd}
	}
	return nil
}

// ---- action helpers ----

// submitTask sends the current input to the agent for processing.
func (m *model) submitTask() tea.Cmd {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return nil
	}
	m.input.Reset()
	m.log = append(m.log,
		compoent.UserMessage{Content: input},
	)
	m.dirty = true
	m.running = true

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	ch := make(chan progressMsg, 64)
	m.ch = ch

	go func(ag *agent.Agent, input string) {
		defer func() {
			if r := recover(); r != nil {
				ch <- progressMsg{err: fmt.Errorf("panic: %v", r)}
				ch <- progressMsg{done: true}
				close(ch)
			}
		}()
		_, err := ag.Run(ctx, input, func(msg agent.CallbackMsg) {
			ch <- progressMsg{
				typ:      msg.Type,
				id:       msg.ID,
				content:  msg.Content,
				toolName: msg.ToolName,
				toolArgs: msg.ToolArgs,
				toolErr:  msg.Err,
			}
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			ch <- progressMsg{err: err}
		}
		ch <- progressMsg{done: true}
		close(ch)
	}(m.agent, input)

	return waitCmd(ch)
}

// cancelAgent cancels the running agent context, stopping the ReAct loop.
func (m *model) cancelAgent() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	// The goroutine will receive context.Canceled, send the error + done
	// messages, and handleProgressMsg will transition out of running state.
}

// applyStreamUpdate finds or creates a streaming component (assistant / thinking)
// and updates its content in-place.
func (m *model) applyStreamUpdate(msg progressMsg) {
	kind := componentTypeStr(msg.typ)
	for i := len(m.log) - 1; i >= 0; i-- {
		if m.log[i].MsgID() == msg.id && m.log[i].Type() == kind {
			switch c := m.log[i].(type) {
			case *compoent.AssistantMessage:
				c.Content = msg.content
			case *compoent.ThinkingMessage:
				c.Content = msg.content
			}
			m.dirty = true
			return
		}
	}
	// Not found — append new streaming component.
	switch kind {
	case "assistant":
		m.log = append(m.log, &compoent.AssistantMessage{ID: msg.id, Content: msg.content})
	case "thinking":
		m.log = append(m.log, &compoent.ThinkingMessage{ID: msg.id, Content: msg.content})
	}
	m.dirty = true
}

// applyToolResult finds the matching tool-call component and sets its result,
// or creates a new one if the call was somehow missed (orphan result).
func (m *model) applyToolResult(msg progressMsg) {
	for i := len(m.log) - 1; i >= 0; i-- {
		if m.log[i].MsgID() == msg.id && m.log[i].Type() == "tool" {
			if tm, ok := m.log[i].(*compoent.ToolMessage); ok {
				tm.SetResult(msg.content)
				if msg.toolErr != nil {
					tm.SetError()
				}
			}
			m.dirty = true
			return
		}
	}
	// Orphan result — create a tool message with the result already set.
	tm := compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs)
	tm.SetResult(msg.content)
	if msg.toolErr != nil {
		tm.SetError()
	}
	m.log = append(m.log, tm)
	m.dirty = true
}

func (m *model) adjustLayout() {
	m.input.SetWidth(m.width - 2)
	m.viewport.SetWidth(m.width - 2)
	m.viewport.SetHeight(max(0, m.height-m.input.Height()-1))
}

func (m *model) updateViewport() {
	// Skip expensive re-render if nothing changed (e.g. during rapid typing/deletion).
	if !m.dirty {
		return
	}
	m.dirty = false

	atBottom := m.viewport.AtBottom()
	var parts []string
	for _, comp := range m.log {
		rendered := comp.Render(m.viewport.Width())
		if rendered != "" {
			parts = append(parts, rendered)
			parts = append(parts, "") // spacing between cards
		}
	}
	content := strings.TrimSpace(strings.Join(parts, "\n"))
	m.viewport.SetContent(content)

	if content != m.lastContent {
		if atBottom {
			m.viewport.GotoBottom()
		}
		m.lastContent = content
	}
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

// componentTypeStr maps agent callback types to component type strings,
// used for finding the right component during streaming updates.
func componentTypeStr(t agent.MsgType) string {
	switch t {
	case agent.MsgThinkingStream:
		return "thinking"
	case agent.MsgAssistantStream:
		return "assistant"
	default:
		return "assistant"
	}
}

// ---- styles (input box only; message styles live in compoent package) ----

var (
	// inputBarStyle — cyan left bar for the input area.
	inputBarStyle = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("14")).
			PaddingLeft(1).
			Foreground(lipgloss.Color("15"))

	// inputBarDimStyle — gray left bar for disabled / processing state.
	inputBarDimStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("8")).
				PaddingLeft(1).
				Foreground(lipgloss.Color("8"))

	// moreLinesStyle — subtle indicator for hidden lines above the input area.
	moreLinesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)
)

func waitCmd(ch chan progressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}
