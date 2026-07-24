package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
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

	st, err := store.Open("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open session store: %v\n", err)
	}
	if st != nil {
		defer st.Close()
	}

	ag := agent.New(apiKey, model, baseURL)
	p := tea.NewProgram(tui.NewModel(ag, st))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
