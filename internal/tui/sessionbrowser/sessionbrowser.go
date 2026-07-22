// Package sessionbrowser provides an interactive session list browser
// built on the bubbles list component, used to select and load past sessions.
package sessionbrowser

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lgzzzz/gocode/internal/store"
)

// ---- sessionItem ----

// sessionItem adapts store.SessionInfo to the list.Item interface.
type sessionItem struct {
	Session store.SessionInfo
}

// FilterValue returns the first user message for filtering (unused: filtering is disabled).
func (si sessionItem) FilterValue() string {
	return si.Session.FirstMsg
}

// ---- styles ----

var (
	sessionItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(1)

	sessionSelectedStyle = sessionItemStyle.
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("4")).
				Background(lipgloss.Color("0"))

	sessionDimStyle = sessionItemStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("8"))

	sessionFirstMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				MaxWidth(80)

	sessionMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				MaxWidth(80)
)

// ---- sessionDelegate ----

// sessionDelegate implements list.ItemDelegate for session items.
type sessionDelegate struct{}

func (d sessionDelegate) Height() int  { return 2 }
func (d sessionDelegate) Spacing() int { return 0 }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(sessionItem)
	if !ok {
		return
	}

	// First line: first user message (truncated)
	firstMsg := si.Session.FirstMsg
	if firstMsg == "" {
		firstMsg = "(empty session)"
	}
	firstMsg = strings.TrimSpace(firstMsg)
	if len(firstMsg) > 100 {
		firstMsg = firstMsg[:97] + "..."
	}

	// Second line: metadata
	t, _ := time.Parse(time.RFC3339, si.Session.CreatedAt)
	timeStr := t.Local().Format("2006-01-02 15:04")
	meta := fmt.Sprintf("%s  %s  %d msgs  %s",
		timeStr, si.Session.Model, si.Session.MessageCount,
		si.Session.CWD)

	// Selected vs dimmed style
	style := sessionDimStyle
	if index == m.Index() {
		style = sessionSelectedStyle
	}

	rendered := style.Render(lipgloss.JoinVertical(lipgloss.Left,
		sessionFirstMsgStyle.Render(firstMsg),
		sessionMetaStyle.Render(meta),
	))

	fmt.Fprint(w, rendered)
}

func (d sessionDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// ---- Browser ----

// Browser wraps a list.Model to provide interactive session selection.
type Browser struct {
	list   list.Model
	active bool
}

// New creates a new session browser component.
func New(width, height int) *Browser {
	delegate := sessionDelegate{}

	// Pre-create an empty list; items are set later via SetSessions.
	l := list.New(nil, delegate, width, height)
	l.Title = "Sessions"
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false)
	l.SetStatusBarItemName("session", "sessions")
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.Styles.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)
	l.KeyMap.Quit.SetEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)

	// Use per-page = max(height / itemHeight, 1) — delegate height 2 + spacing 0
	perPage := height / 2
	if perPage < 1 {
		perPage = 1
	}
	l.Paginator.PerPage = perPage

	return &Browser{list: l, active: true}
}

// SetActive sets the active state of the browser.
func (b *Browser) SetActive(active bool) {
	b.active = active
}

// Active returns whether the browser is currently active.
func (b *Browser) Active() bool {
	return b.active
}

// SetSessions populates the browser with the given sessions.
func (b *Browser) SetSessions(sessions []store.SessionInfo) {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{Session: s}
	}
	b.list.SetItems(items)
}

// Selected returns the currently selected session, or nil if none.
func (b *Browser) Selected() *store.SessionInfo {
	item := b.list.SelectedItem()
	if item == nil {
		return nil
	}
	si, ok := item.(sessionItem)
	if !ok {
		return nil
	}
	s := si.Session
	return &s
}

// Update delegates to the underlying list model.
func (b *Browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newList, cmd := b.list.Update(msg)
	b.list = newList
	return nil, cmd
}

// View returns the rendered list.
func (b *Browser) View() string {
	return b.list.View()
}

// SetSize updates the browser dimensions and recalculates pagination.
func (b *Browser) SetSize(width, height int) {
	b.list.SetSize(width, height)
}
