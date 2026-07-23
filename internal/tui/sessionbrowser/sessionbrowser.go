// Package sessionbrowser provides an interactive session list browser
// built on the bubbles list component, used to select and load past sessions.
package sessionbrowser

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lgzzzz/gocode/internal/store"
)

// ---- SessionStore ----

// SessionStore is the storage abstraction required by Browser.
// The concrete store.Store satisfies this interface automatically.
type SessionStore interface {
	ListSessions(limit int) ([]store.SessionInfo, error)
	GetSessionMessages(sessionID string) ([]store.Message, error)
}

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
)

// ---- sessionDelegate ----

// sessionDelegate implements list.ItemDelegate for session items.
type sessionDelegate struct{}

func (d sessionDelegate) Height() int  { return 1 }
func (d sessionDelegate) Spacing() int { return 0 }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(sessionItem)
	if !ok {
		return
	}

	// Time prefix.
	t, _ := time.Parse(time.RFC3339, si.Session.CreatedAt)
	timeStr := t.Local().Format("2006-01-02 15:04")

	// First user message (trimmed).
	firstMsg := si.Session.FirstMsg
	if firstMsg == "" {
		firstMsg = "(empty session)"
	}
	firstMsg = strings.TrimSpace(firstMsg)

	// Selected vs dimmed style.
	style := sessionDimStyle
	if index == m.Index() {
		style = sessionSelectedStyle
	}

	line := timeStr + "  " + firstMsg

	blockWidth := m.Width()
	textWidth := blockWidth - style.GetHorizontalPadding() - style.GetHorizontalBorderSize()
	if textWidth < 1 {
		textWidth = 1
	}

	line = ansi.Truncate(line, textWidth, "")
	rendered := style.Width(blockWidth).Render(line)

	fmt.Fprint(w, rendered)
}

func (d sessionDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// ---- Browser ----

// Browser wraps a list.Model to provide interactive session selection.
// It queries sessions from the injected SessionStore on activation.
type Browser struct {
	list   list.Model
	store  SessionStore
	active bool
}

// New creates a new session browser component. The store may be nil
// if persistence is unavailable — in that case Reload will return
// an error. Does not load data; call Reload or SetSessions to
// populate the list.
func New(width, height int, store SessionStore) *Browser {
	delegate := sessionDelegate{}

	l := list.New(nil, delegate, width, height)
	l.Title = "Sessions"
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false)
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.Styles.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)
	l.KeyMap.Quit.SetEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)

	// Use per-page = height (delegate height 1 + spacing 0)
	perPage := height
	if perPage < 1 {
		perPage = 1
	}
	l.Paginator.PerPage = perPage

	return &Browser{list: l, store: store}
}

// ---- state ----

// SetActive sets the active state of the browser.
func (b *Browser) SetActive(active bool) {
	b.active = active
}

// Active returns whether the browser is currently active.
func (b *Browser) Active() bool {
	return b.active
}

// IsEmpty reports whether the session list is empty.
func (b *Browser) IsEmpty() bool {
	return len(b.list.Items()) == 0
}

// ---- data ----

// Reload re-queries sessions from the store and refreshes the list.
// Returns an error if the store is nil or the query fails.
func (b *Browser) Reload() error {
	if b.store == nil {
		return errors.New("session store is unavailable")
	}
	sessions, err := b.store.ListSessions(50)
	if err != nil {
		return err
	}
	b.SetSessions(sessions)
	return nil
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

// GetMessages returns all persisted messages for the given session.
func (b *Browser) GetMessages(sessionID string) ([]store.Message, error) {
	return b.store.GetSessionMessages(sessionID)
}

// ---- bubbletea.Model ----

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
