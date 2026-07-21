package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	ch       chan progressMsg

	moreLines   int    // number of lines hidden above the input area viewport
	lastContent string // track content to detect when it changes (for auto-scroll)
}

// NewModel creates a new TUI model.
func NewModel(ag *agent.Agent) tea.Model {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width, height = 80, 24
	}

	ta := textarea.New()
	ta.Placeholder = "Describe your coding task..."
	ta.Focus()
	ta.ShowLineNumbers = false
	vp := viewport.New(0, 0) // account for input area and padding
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "insert new line"))

	m := model{
		input:    ta,
		viewport: vp,
		agent:    ag,
		width:    width,
		height:   height,
	}
	m.adjustInputHeight()
	return m
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		cmds = append(cmds, m.handleMouseMsg(msg)...)
	case tea.KeyMsg:
		cmds = append(cmds, m.handleKeyMsg(msg)...)
	case tea.WindowSizeMsg:
		cmds = append(cmds, m.handleWindowSizeMsg(msg)...)
	case progressMsg:
		cmds = append(cmds, m.handleProgressMsg(msg)...)
	}

	m.adjustInputHeight()
	m.updateViewport()

	return m, tea.Batch(cmds...)
}

// ---- message handlers ----

// handleMouseMsg routes mouse events to the appropriate sub-components.
// Wheel events go to viewport only (for scrolling); click/motion goes to both.
func (m *model) handleMouseMsg(msg tea.MouseMsg) []tea.Cmd {
	if isMouseWheel(msg) {
		return m.updateViewportModel(msg)
	}
	return nil
}

// handleKeyMsg processes keyboard events: arrow keys go to input only,
// other keys go to both input and viewport, plus special key bindings.
func (m *model) handleKeyMsg(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
		cmds = append(cmds, m.updateInput(msg)...)
	} else {
		cmds = append(cmds, m.updateInput(msg)...)
	}

	// Special key bindings (quit, submit, etc.)
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		cmds = append(cmds, tea.Quit)
		return cmds

	case tea.KeyEnter:
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
		return nil
	}

	if msg.done {
		m.running = false
		m.ch = nil
		return nil
	}

	switch msg.typ {
	case agent.MsgAssistantStream, agent.MsgThinkingStream:
		m.applyStreamUpdate(msg)

	case agent.MsgToolCall:
		m.log = append(m.log, compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs))

	case agent.MsgToolResult:
		m.applyToolResult(msg)

	default:
		m.log = append(m.log, &compoent.AssistantMessage{ID: msg.id, Content: msg.content})
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
	m.running = true

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
		_, err := ag.Run(context.Background(), input, func(msg agent.CallbackMsg) {
			ch <- progressMsg{
				typ:      msg.Type,
				id:       msg.ID,
				content:  msg.Content,
				toolName: msg.ToolName,
				toolArgs: msg.ToolArgs,
				toolErr:  msg.Err,
			}
		})
		if err != nil {
			ch <- progressMsg{err: err}
		}
		ch <- progressMsg{done: true}
		close(ch)
	}(m.agent, input)

	return waitCmd(ch)
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
}

func (m *model) adjustInputHeight() {
	totalLines := len(strings.Split(m.input.View(), "\n"))
	maxVisible := 7

	// Always clamp input height to actual content lines, capped at maxVisible.
	// This ensures the input area shrinks when content is deleted.
	m.input.SetHeight(min(totalLines, maxVisible))
	if totalLines > maxVisible {
		m.moreLines = totalLines - maxVisible
	} else {
		m.moreLines = 0
	}

	m.input.SetWidth(m.width - 2)

	// Dynamically adjust viewport height when input height changes.
	if m.height > 0 {
		// Reserve space for: input area (visible lines) + 1 gap/more-indicator line.
		inputReserved := m.input.Height() + 1
		m.viewport.Height = max(0, m.height-inputReserved)
		m.viewport.Width = m.width - 2
	}
}

func (m *model) updateViewport() {
	var parts []string
	for _, comp := range m.log {
		rendered := comp.Render(m.viewport.Width)
		if rendered != "" {
			parts = append(parts, rendered)
			parts = append(parts, "") // spacing between cards
		}
	}
	content := strings.TrimSpace(strings.Join(parts, "\n"))
	m.viewport.SetContent(content)

	if content != m.lastContent {
		m.viewport.GotoBottom()
		m.lastContent = content
	}
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var inputArea string
	if m.running {
		inputArea = inputBarDimStyle.Render("⏳ Processing... (please wait)")
	} else {
		inputArea = m.input.View()
	}

	// Show "X more lines/line" indicator when input exceeds visible area.
	var moreIndicator string
	if m.moreLines > 0 {
		if m.moreLines == 1 {
			moreIndicator = moreLinesStyle.Render("1 more line")
		} else {
			moreIndicator = moreLinesStyle.Render(fmt.Sprintf("%d more lines", m.moreLines))
		}
	}

	// Put input area inside the viewport at the bottom
	vpContent := m.viewport.View()
	return lipgloss.JoinVertical(lipgloss.Left,
		vpContent,
		moreIndicator,
		inputArea,
	)
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

// isMouseWheel reports whether a mouse event is a wheel/scroll action.
func isMouseWheel(m tea.MouseMsg) bool {
	return m.Button == tea.MouseButtonWheelUp ||
		m.Button == tea.MouseButtonWheelDown ||
		m.Button == tea.MouseButtonWheelLeft ||
		m.Button == tea.MouseButtonWheelRight
}

func waitCmd(ch chan progressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}
