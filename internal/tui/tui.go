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
	content string
}

// ---- progress message ----

type progressMsg struct {
	text string
	done bool
	err  error
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
	ta.Prompt = "> "
	ta.CharLimit = 0

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
		m.input.SetWidth(msg.Width - 6)
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
					_, err := ag.Run(context.Background(), input, func(text string) {
						ch <- progressMsg{text: text}
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
			kind := classify(msg.text)
			// strip the prefix for clean card content
			content := msg.text
			switch kind {
			case msgThinking:
				content = strings.TrimPrefix(content, "💭 ")
			case msgAssistant:
				content = strings.TrimPrefix(content, "🤖 ")
			case msgToolCall:
				content = strings.TrimPrefix(content, "🔧 ")
			case msgToolResult:
				content = strings.TrimPrefix(content, "→ ")
			}
			m.log = append(m.log,
				message{kind: kind, content: content},
			)
		}
		m.updateViewport()

		if !msg.done && m.ch != nil {
			cmds = append(cmds, waitCmd(m.ch))
		}
	}

	return m, tea.Batch(cmds...)
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
		inputArea = runStyle.Render(" ⏳ Processing... (please wait)")
	} else {
		inputArea = m.input.View()
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
	innerW := width - 4 // account for borders + padding
	if innerW < 20 {
		innerW = 20
	}

	switch msg.kind {

	case msgUser:
		header := userHeaderStyle.Render("🧑 You")
		body := userBodyStyle.Width(innerW).Render(wrapText(msg.content, innerW))
		return lipgloss.JoinVertical(lipgloss.Left, header, body)

	case msgThinking:
		header := thinkHeaderStyle.Render("💭 Thinking")
		body := thinkBodyStyle.Width(innerW).Render(wrapText(msg.content, innerW))
		return lipgloss.JoinVertical(lipgloss.Left, header, body)

	case msgAssistant:
		header := asstHeaderStyle.Render("🤖 Assistant")
		body := asstBodyStyle.Width(innerW).Render(wrapText(msg.content, innerW))
		return lipgloss.JoinVertical(lipgloss.Left, header, body)

	case msgToolCall:
		header := toolCallHeaderStyle.Render("🔧 Tool Call")
		body := toolCallBodyStyle.Width(innerW).Render(wrapText(msg.content, innerW))
		return lipgloss.JoinVertical(lipgloss.Left, header, body)

	case msgToolResult:
		header := resultHeaderStyle.Render("📋 Result")
		body := resultBodyStyle.Width(innerW).Render(wrapText(msg.content, innerW))
		return lipgloss.JoinVertical(lipgloss.Left, header, body)

	case msgError:
		header := errHeaderStyle.Render("❌ Error")
		body := errBodyStyle.Width(innerW).Render(wrapText(msg.content, innerW))
		return lipgloss.JoinVertical(lipgloss.Left, header, body)

	case msgDone:
		return doneCardStyle.Width(width).Render(" ✅ " + msg.content + " ✅ ")

	case msgSystem:
		return systemCardStyle.Width(width).Render(msg.content)

	default:
		return msg.content
	}
}

// classify determines the message kind from the callback text prefix.
func classify(text string) msgType {
	switch {
	case strings.HasPrefix(text, "🧑"):
		return msgUser
	case strings.HasPrefix(text, "💭"):
		return msgThinking
	case strings.HasPrefix(text, "🤖"):
		return msgAssistant
	case strings.HasPrefix(text, "🔧"):
		return msgToolCall
	case strings.HasPrefix(text, "   →"):
		return msgToolResult
	case strings.HasPrefix(text, "❌"):
		return msgError
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

// User card
var (
	userHeaderStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Background(colorCyan).
			Bold(true).
			Padding(0, 1)

	userBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(colorCyan).
			Foreground(lipgloss.Color("14")).
			Padding(0, 1)
)

// Thinking card
var (
	thinkHeaderStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorPurple).
				Bold(true).
				Padding(0, 1)

	thinkBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(colorPurple).
			Foreground(lipgloss.Color("13")).
			Italic(true).
			Padding(0, 1)
)

// Assistant card
var (
	asstHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(colorGreen).
			Bold(true).
			Padding(0, 1)

	asstBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(colorGreen).
			Foreground(lipgloss.Color("2")).
			Padding(0, 1)
)

// Tool Call card
var (
	toolCallHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(colorYellow).
				Bold(true).
				Padding(0, 1)

	toolCallBodyStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true, true, true).
				BorderForeground(colorYellow).
				Foreground(lipgloss.Color("11")).
				Padding(0, 1)
)

// Tool Result card
var (
	resultHeaderStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorGray).
				Bold(true).
				Padding(0, 1)

	resultBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(colorGray).
			Foreground(lipgloss.Color("7")).
			Padding(0, 1)
)

// Error card
var (
	errHeaderStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Background(colorRed).
			Bold(true).
			Padding(0, 1)

	errBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(colorRed).
			Foreground(lipgloss.Color("9")).
			Padding(0, 1)
)

// Done card
var doneCardStyle = lipgloss.NewStyle().
	Foreground(colorGreen).
	Bold(true).
	Align(lipgloss.Center).
	Border(lipgloss.NormalBorder(), false, true, false, true).
	BorderForeground(colorGreen).
	Padding(0, 1)

// System / welcome card
var systemCardStyle = lipgloss.NewStyle().
	Align(lipgloss.Center).
	Foreground(colorPurple).
	Bold(true).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorPurple).
	Padding(1, 2)

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
