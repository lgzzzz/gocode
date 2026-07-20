package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lgzzzz/gocode/internal/agent"
)

// ---- message types ----

type msgType int

const (
	msgUser       msgType = iota // user input
	msgThinking                  // assistant reasoning/thinking
	msgAssistant                 // final assistant response
	msgToolCall                  // tool being invoked
	msgToolResult                // tool execution result
	msgError                     // error message
	msgDone                      // task complete
	msgSystem                    // system / welcome banner
)

type message struct {
	kind    msgType
	id      string // message ID for in-place streaming updates
	content string
}

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
	log      []message
	width    int
	height   int
	running  bool
	ch       chan progressMsg
}

// NewModel creates a new TUI model.
func NewModel(ag *agent.Agent) tea.Model {
	ta := textarea.New()
	ta.Placeholder = "Describe your coding task..."
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.MaxHeight = 5

	vp := viewport.New(80, 20)

	return model{
		input:    ta,
		viewport: vp,
		agent:    ag,
		log: []message{
			{kind: msgSystem, content: "🤖 AI Coding Agent — ReAct + DeepSeek · Go"},
		},
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// update sub-components
	newInput, inputCmd := m.input.Update(msg)
	m.input = newInput

	// Dynamically adjust input height based on content (3-5 rows).
	m.adjustInputHeight()

	newVP, vpCmd := m.viewport.Update(msg)
	m.viewport = newVP

	var cmds []tea.Cmd
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
		m.input.SetWidth(msg.Width - 8) // account for border + padding
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - 5 // leave room for input area inside viewport

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyCtrlD:
			if !m.running {
				input := strings.TrimSpace(m.input.Value())
				if input == "" {
					break
				}
				m.input.Reset()
				m.log = append(m.log,
					message{kind: msgUser, content: input},
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
				message{kind: msgError, content: msg.err.Error()},
			)
		} else if msg.done {
			m.running = false
			m.ch = nil
			m.log = append(m.log,
				message{kind: msgDone, content: "── Done ──"},
			)
		} else {
			switch msg.typ {
			case agent.MsgAssistantStream, agent.MsgThinkingStream:
				// Streaming update: find existing message by ID and update in-place,
				// or append a new one if this is the first delta.
				kind := msgTypeForCallback(msg.typ)
				found := false
				for i := len(m.log) - 1; i >= 0; i-- {
					if m.log[i].id == msg.id && m.log[i].kind == kind {
						m.log[i].content = msg.content
						found = true
						break
					}
				}
				if !found {
					m.log = append(m.log, message{
						kind:    kind,
						id:      msg.id,
						content: msg.content,
					})
				}

			case agent.MsgThinking:
				// Non-streaming thinking block — append as new message.
				m.log = append(m.log, message{
					kind:    msgThinking,
					id:      msg.id,
					content: msg.content,
				})

			case agent.MsgToolCall:
				m.log = append(m.log, message{
					kind:    msgToolCall,
					id:      msg.id,
					content: fmt.Sprintf("%s(%s)", msg.toolName, msg.toolArgs),
				})

			case agent.MsgToolResult:
				m.log = append(m.log, message{
					kind:    msgToolResult,
					id:      msg.id,
					content: msg.content,
				})

			case agent.MsgAssistantDone:
				// Streaming finished — mark the assistant message as done (no-op for now,
				// the last streaming delta already has the full content).

			default:
				// Fallback: treat as assistant content
				m.log = append(m.log, message{
					kind:    msgAssistant,
					id:      msg.id,
					content: msg.content,
				})
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
	lines := m.input.LineCount()
	h := 3
	if lines > 3 {
		h = lines
	}
	if h > 5 {
		h = 5
	}
	m.input.SetHeight(h)
}

func (m *model) updateViewport() {
	var parts []string
	for _, msg := range m.log {
		parts = append(parts, renderCard(msg, m.viewport.Width))
		parts = append(parts, "") // spacing between cards
	}
	m.viewport.SetContent(strings.Join(parts, "\n"))
	m.viewport.GotoBottom()
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var inputArea string
	if m.running {
		inputArea = inputBoxDimStyle.Render(runStyle.Render(" ⏳ Processing... (please wait)"))
	} else {
		inputArea = renderInputBox(m.input)
	}

	// Put input area inside the viewport at the bottom
	vpContent := m.viewport.View()
	return lipgloss.JoinVertical(lipgloss.Left,
		vpContent,
		inputArea,
	)
}

// ---- card rendering ----

func cardLayout(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		PaddingLeft(1).
		PaddingRight(1)
}

func renderCard(msg message, width int) string {
	innerW := width - 6 // account for rounded borders + padding + margins
	if innerW < 20 {
		innerW = 20
	}

	// Helper to produce: [emoji label] + [colored "▌" bar + body text]
	renderBarCard := func(label, emoji string, cardStyle, labelStyle lipgloss.Style, content string) string {
		labelLine := labelStyle.Render(emoji + " " + label)
		body := cardStyle.Width(innerW).Render(wrapText(content, innerW))
		return lipgloss.JoinVertical(lipgloss.Left, labelLine, body)
	}

	switch msg.kind {

	case msgUser:
		return renderBarCard("You", "🧑", userCardStyle, userLabelStyle, msg.content)

	case msgThinking:
		return renderBarCard("Thinking", "💭", thinkCardStyle, thinkLabelStyle, msg.content)

	case msgAssistant:
		return renderBarCard("Assistant", "🤖", asstCardStyle, asstLabelStyle, msg.content)

	case msgToolCall:
		return renderBarCard("Tool Call", "🔧", toolCallCardStyle, toolCallLabelStyle, msg.content)

	case msgToolResult:
		return renderBarCard("Result", "📋", resultCardStyle, resultLabelStyle, msg.content)

	case msgError:
		return renderBarCard("Error", "❌", errCardStyle, errLabelStyle, msg.content)

	case msgDone:
		return doneCardStyle.Width(width).Render(" ✅ " + msg.content + " ✅ ")

	case msgSystem:
		return systemCardStyle.Width(width).Render(msg.content)

	default:
		return msg.content
	}
}

// msgTypeForCallback maps agent.CallbackMsg types to TUI message kinds.
func msgTypeForCallback(t agent.MsgType) msgType {
	switch t {
	case agent.MsgThinkingStream, agent.MsgThinking:
		return msgThinking
	case agent.MsgAssistantStream:
		return msgAssistant
	case agent.MsgToolCall:
		return msgToolCall
	case agent.MsgToolResult:
		return msgToolResult
	default:
		return msgAssistant
	}
}

// wrapText does simple word wrapping for card bodies.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		if len(paragraph) <= width {
			lines = append(lines, paragraph)
			continue
		}
		// word-wrap long lines
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := words[0]
		for _, w := range words[1:] {
			if len(current)+1+len(w) <= width {
				current += " " + w
			} else {
				lines = append(lines, current)
				current = w
			}
		}
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}

// ---- styles ----
//
// Modern left accent bar design: each card type uses a colored "▌" half-block
// bar on the left side (via lipgloss.OuterHalfBlockBorder + BorderLeft only).
// Color-coding makes message sources instantly recognizable.

var (
	colorCyan   = lipgloss.Color("6")
	colorGreen  = lipgloss.Color("10")
	colorYellow = lipgloss.Color("3")
	colorRed    = lipgloss.Color("9")
	colorPurple = lipgloss.Color("13")
	colorGray   = lipgloss.Color("8")
	colorWhite  = lipgloss.Color("15")
)

var runStyle = lipgloss.NewStyle().Foreground(colorYellow).Italic(true)

// leftBarCard creates a style with only a colored left "▌" bar.
func leftBarCard(accent lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.OuterHalfBlockBorder()). // "▌" half-block
		BorderLeftForeground(accent).
		PaddingLeft(1)
}

// User card — pushed right via left margin
var (
	userCardStyle = leftBarCard(colorCyan).
			Foreground(lipgloss.Color("14")).
			MarginLeft(6)

	userLabelStyle = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true).
			Padding(0, 1)
)

// Thinking card
var (
	thinkCardStyle = leftBarCard(colorPurple).
			Foreground(lipgloss.Color("13")).
			Italic(true).
			MarginRight(6)

	thinkLabelStyle = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true).
			Padding(0, 1)
)

// Assistant card
var (
	asstCardStyle = leftBarCard(colorGreen).
			Foreground(lipgloss.Color("15")).
			MarginRight(6)

	asstLabelStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true).
			Padding(0, 1)
)

// Tool Call card
var (
	toolCallCardStyle = leftBarCard(colorYellow).
			Foreground(lipgloss.Color("11")).
			MarginRight(6)

	toolCallLabelStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true).
			Padding(0, 1)
)

// Tool Result card
var (
	resultCardStyle = leftBarCard(colorGray).
			Foreground(lipgloss.Color("7")).
			MarginRight(6)

	resultLabelStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Bold(true).
			Padding(0, 1)
)

// Error card
var (
	errCardStyle = leftBarCard(colorRed).
			Foreground(lipgloss.Color("9")).
			Bold(true).
			MarginRight(6)

	errLabelStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true).
			Padding(0, 1)
)

// Done card — centered, no left bar
var doneCardStyle = lipgloss.NewStyle().
	Foreground(colorGreen).
	Bold(true).
	Align(lipgloss.Center).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorGreen).
	Padding(0, 1)

// System / welcome card — centered rounded banner
var systemCardStyle = lipgloss.NewStyle().
	Align(lipgloss.Center).
	Foreground(colorPurple).
	Bold(true).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorPurple).
	Padding(1, 2)

// ---- input box rendering ----

var (
	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(0, 1).
			MarginTop(1)

	inputBoxDimStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("8")).
				Padding(0, 1).
				MarginTop(1)
)

// renderInputBox renders the textarea inside a rounded border.
// When the content has more logical lines than the visible height,
// the bottom border shows "N more lines/line".
func renderInputBox(ta textarea.Model) string {
	content := ta.View()
	overflow := ta.LineCount() - ta.Height()
	if overflow <= 0 {
		return inputBoxStyle.Render(content)
	}

	// Build overflow label
	label := fmt.Sprintf(" %d more lines ", overflow)
	if overflow == 1 {
		label = " 1 more line "
	}

	// Render with the normal border, then patch the bottom line.
	boxed := inputBoxStyle.Render(content)
	lines := strings.Split(boxed, "\n")
	if len(lines) < 2 {
		return boxed
	}

	// Rebuild bottom border with label embedded.
	border := lipgloss.RoundedBorder()
	leftCorner := border.BottomLeft
	rightCorner := border.BottomRight
	hbar := border.Bottom

	topLine := lines[0]
	innerWidth := lipgloss.Width(topLine) - lipgloss.Width(leftCorner) - lipgloss.Width(rightCorner)
	if innerWidth < lipgloss.Width(label) {
		innerWidth = lipgloss.Width(label)
	}

	labelW := lipgloss.Width(label)
	leftPad := (innerWidth - labelW) / 2
	rightPad := innerWidth - labelW - leftPad

	customBottom := leftCorner + strings.Repeat(hbar, leftPad) + label + strings.Repeat(hbar, rightPad) + rightCorner

	// Apply border foreground color.
	noColor := lipgloss.NoColor{}
	fg := inputBoxStyle.GetBorderTopForeground()
	if fg != noColor {
		customBottom = lipgloss.NewStyle().Foreground(fg).Render(customBottom)
	}

	lines[len(lines)-1] = customBottom
	return strings.Join(lines, "\n")
}

// ---- channel wait command ----

func waitCmd(ch chan progressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}
