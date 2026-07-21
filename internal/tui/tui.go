package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	done     bool
	err      error
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

	// update sub-components
	newInput, inputCmd := m.input.Update(msg)
	m.input = newInput

	// Dynamically adjust input height based on content (1-3 rows).
	m.adjustInputHeight()

	newVP, vpCmd := m.viewport.Update(msg)
	m.viewport = newVP

	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyShiftTab:
			if !m.running {
				m.input.InsertString("\n")
			}

		case tea.KeyEnter:
			if !m.running {
				input := strings.TrimSpace(m.input.Value())
				if input == "" {
					break
				}
				m.input.Reset()
				m.log = append(m.log,
					compoent.UserMessage{Content: input},
				)
				m.updateViewport()
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
						}
					})
					if err != nil {
						ch <- progressMsg{err: err}
					}
					ch <- progressMsg{done: true}
					close(ch)
				}(m.agent, input)

				cmds = append(cmds, waitCmd(ch))
			}
		}

	case progressMsg:
		if msg.err != nil {
			m.log = append(m.log,
				compoent.ErrorMessage{Content: msg.err.Error()},
			)
		} else if msg.done {
			m.running = false
			m.ch = nil
		} else {
			switch msg.typ {
			case agent.MsgAssistantStream, agent.MsgThinkingStream:
				// Streaming update: find existing component by ID+type and update in-place,
				// or append a new one if this is the first delta.
				kind := componentTypeStr(msg.typ)
				found := false
				for i := len(m.log) - 1; i >= 0; i-- {
					if m.log[i].MsgID() == msg.id && m.log[i].Type() == kind {
						switch c := m.log[i].(type) {
						case *compoent.AssistantMessage:
							c.Content = msg.content
						case *compoent.ThinkingMessage:
							c.Content = msg.content
						}
						found = true
						break
					}
				}
				if !found {
					switch kind {
					case "assistant":
						m.log = append(m.log, &compoent.AssistantMessage{ID: msg.id, Content: msg.content})
					case "thinking":
						m.log = append(m.log, &compoent.ThinkingMessage{ID: msg.id, Content: msg.content})
					}
				}

			case agent.MsgThinking:
				m.log = append(m.log, &compoent.ThinkingMessage{ID: msg.id, Content: msg.content})

			case agent.MsgToolCall:
				m.log = append(m.log, compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs))

			case agent.MsgToolResult:
				found := false
				for i := len(m.log) - 1; i >= 0; i-- {
					if m.log[i].MsgID() == msg.id && m.log[i].Type() == "tool" {
						if tm, ok := m.log[i].(*compoent.ToolMessage); ok {
							tm.SetResult(msg.content)
						}
						found = true
						break
					}
				}
				if !found {
					// Orphan result — create a tool message with the result already set.
					tm := compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs)
					tm.SetResult(msg.content)
					m.log = append(m.log, tm)
				}

			case agent.MsgAssistantDone:
				// Streaming finished — the last streaming delta already has the full content.

			default:
				m.log = append(m.log, &compoent.AssistantMessage{ID: msg.id, Content: msg.content})
			}
		}
		m.updateViewport()

		if !msg.done && m.ch != nil {
			cmds = append(cmds, waitCmd(m.ch))
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *model) adjustInputHeight() {
	m.input.SetHeight(1)
	m.input.SetWidth(m.width - 2)
	// Dynamically adjust viewport height when input height changes.
	if m.height > 0 {
		// Reserve space for input area: input height + 2 for padding/border.
		m.viewport.Height = m.height - 2
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
	m.viewport.SetContent(strings.TrimSpace(strings.Join(parts, "\n")))
	m.viewport.GotoBottom()
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

	// Put input area inside the viewport at the bottom
	vpContent := m.viewport.View()
	return lipgloss.JoinVertical(lipgloss.Left,
		vpContent,
		"",
		inputArea,
	)
}

// componentTypeStr maps agent callback types to component type strings,
// used for finding the right component during streaming updates.
func componentTypeStr(t agent.MsgType) string {
	switch t {
	case agent.MsgThinkingStream, agent.MsgThinking:
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
