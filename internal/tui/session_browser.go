package tui

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

// ---- SessionItem ----

// SessionItem adapts store.SessionInfo to the list.Item interface.
type SessionItem struct {
	Session          store.SessionInfo
	currentSessionID string
}

// FilterValue returns the first user message for filtering (unused: filtering is disabled).
func (si SessionItem) FilterValue() string {
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

	sessionCurrentStyle = sessionItemStyle.
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("11")) // yellow

	sessionFirstMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				MaxWidth(80)

	sessionMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				MaxWidth(80)
)

// ---- SessionDelegate ----

// SessionDelegate implements list.ItemDelegate for session items.
type SessionDelegate struct{}

func (d SessionDelegate) Height() int   { return 2 }
func (d SessionDelegate) Spacing() int  { return 0 }

func (d SessionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(SessionItem)
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
	marker := ""
	if si.Session.ID == si.currentSessionID {
		marker = " ◀ current"
	}
	meta := fmt.Sprintf("%s  %s  %d msgs  %s%s",
		timeStr, si.Session.Model, si.Session.MessageCount,
		si.Session.CWD, marker)

	// Determine style based on selection and current status
	style := sessionDimStyle
	isCurrent := si.Session.ID == si.currentSessionID
	isSelected := index == m.Index()

	switch {
	case isCurrent && isSelected:
		style = sessionCurrentStyle.Background(lipgloss.Color("0"))
	case isCurrent:
		style = sessionCurrentStyle
	case isSelected:
		style = sessionSelectedStyle
	}

	rendered := style.Render(lipgloss.JoinVertical(lipgloss.Left,
		sessionFirstMsgStyle.Render(firstMsg),
		sessionMetaStyle.Render(meta),
	))

	fmt.Fprint(w, rendered)
}

func (d SessionDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// ---- SessionBrowser ----

// SessionBrowser wraps a list.Model to provide interactive session selection.
type SessionBrowser struct {
	list  list.Model
	items []SessionItem
}

// NewSessionBrowser creates a new session browser component.
// currentSessionID is used to highlight the currently active session.
func NewSessionBrowser(width, height int, currentSessionID string) *SessionBrowser {
	delegate := SessionDelegate{}

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

	return &SessionBrowser{list: l}
}

// SetSessions populates the browser with the given sessions.
func (sb *SessionBrowser) SetSessions(sessions []store.SessionInfo, currentSessionID string) {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = SessionItem{Session: s, currentSessionID: currentSessionID}
	}
	sb.items = make([]SessionItem, len(items))
	for i, item := range items {
		sb.items[i] = item.(SessionItem)
	}
	sb.list.SetItems(items)
}

// Selected returns the currently selected session, or nil if none.
func (sb *SessionBrowser) Selected() *store.SessionInfo {
	item := sb.list.SelectedItem()
	if item == nil {
		return nil
	}
	si, ok := item.(SessionItem)
	if !ok {
		return nil
	}
	s := si.Session
	return &s
}

// Update delegates to the underlying list model.
func (sb *SessionBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newList, cmd := sb.list.Update(msg)
	sb.list = newList
	return nil, cmd
}

// View returns the rendered list.
func (sb *SessionBrowser) View() string {
	return sb.list.View()
}

// SetSize updates the browser dimensions and recalculates pagination.
func (sb *SessionBrowser) SetSize(width, height int) {
	sb.list.SetSize(width, height)
	perPage := height / 2
	if perPage < 1 {
		perPage = 1
	}
	sb.list.Paginator.PerPage = perPage
}
