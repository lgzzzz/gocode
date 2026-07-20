package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui"
)

func main() {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: DEEPSEEK_API_KEY environment variable is not set.")
		fmt.Fprintln(os.Stderr, "Usage: export DEEPSEEK_API_KEY=sk-...")
		os.Exit(1)
	}

	model := "deepseek-v4-pro"
	baseURL := "https://api.deepseek.com"

	ag := agent.New(apiKey, model, baseURL)
	p := tea.NewProgram(tui.NewModel(ag), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
