package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
)

// ---- progress message ----

type progressMsg struct {
	msg  agent.CallbackMsg
	done bool
	err  error // fatal / panic error
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
