package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---- messages ----

type progressMsg struct {
	text string
	done bool
	err  error
}

// ---- model ----

type tuiModel struct {
	input    textarea.Model
	viewport viewport.Model
	agent    *Agent
	log      []string
	width    int
	height   int
	running  bool
	ch       chan progressMsg
}

func newTUIModel(agent *Agent) tuiModel {
	ta := textarea.New()
	ta.Placeholder = "Describe your coding task..."
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.Prompt = "> "
	ta.CharLimit = 0

	vp := viewport.New(80, 20)
	vp.SetContent(strings.Join([]string{
		logoStyle.Render("╔══════════════════════════════════╗"),
		logoStyle.Render("║   🤖 AI Coding Agent           ║"),
		logoStyle.Render("║   ReAct + DeepSeek  ·  Go     ║"),
		logoStyle.Render("╚══════════════════════════════════╝"),
		"",
		"  Tools: read · write · edit · bash",
		"  Ctrl+D submit  ·  Ctrl+C quit",
		"",
	}, "\n"))

	return tuiModel{
		input:    ta,
		viewport: vp,
		agent:    agent,
		log:      []string{},
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// --- update sub-components first ---
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

	// --- custom handling ---
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - 6)
		// override viewport dims (viewport.Update already set them to raw window size)
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - 7

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
					userStyle.Render("🧑 You:"),
					"   "+input,
					"",
				)
				m.updateViewport()
				m.running = true

				ch := make(chan progressMsg, 64)
				m.ch = ch

				go func(agent *Agent, input string) {
					defer func() {
						if r := recover(); r != nil {
							ch <- progressMsg{err: fmt.Errorf("panic: %v", r)}
							ch <- progressMsg{done: true}
							close(ch)
						}
					}()
					_, err := agent.Run(context.Background(), input, func(text string) {
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
			m.log = append(m.log, errStyle.Render("❌ "+msg.err.Error()))
		} else if msg.done {
			m.running = false
			m.ch = nil
			m.log = append(m.log, "", doneStyle.Render("── Done ──"), "")
		} else {
			m.log = append(m.log, msg.text)
		}
		m.updateViewport()

		if !msg.done && m.ch != nil {
			cmds = append(cmds, waitCmd(m.ch))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *tuiModel) updateViewport() {
	m.viewport.SetContent(strings.Join(m.log, "\n"))
	m.viewport.GotoBottom()
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	sep := strings.Repeat("─", m.width-2)

	var inputArea string
	if m.running {
		inputArea = runStyle.Render(" ⏳ Processing... (please wait)")
	} else {
		inputArea = m.input.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		sepStyle.Render(sep),
		inputArea,
	)
}

// ---- styles ----

var (
	userStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	doneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	sepStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	runStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Italic(true)
	logoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
)

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

// ---- entry ----

func main() {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: DEEPSEEK_API_KEY environment variable is not set.")
		fmt.Fprintln(os.Stderr, "Usage: export DEEPSEEK_API_KEY=sk-...")
		os.Exit(1)
	}

	model := "deepseek-v4-pro"
	baseURL := "https://api.deepseek.com"

	agent := NewAgent(apiKey, model, baseURL)
	p := tea.NewProgram(newTUIModel(agent), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
