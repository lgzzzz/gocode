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


type SessionStore interface {
	ListSessions(limit int) ([]store.Session, error)
	GetSessionMessages(sessionID string) ([]store.Message, error)
}


type sessionItem struct {
	Session store.Session
}

func (si sessionItem) FilterValue() string {
	return si.Session.FirstMsg
}


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


type sessionDelegate struct{}

func (d sessionDelegate) Height() int  { return 1 }
func (d sessionDelegate) Spacing() int { return 0 }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(sessionItem)
	if !ok {
		return
	}

	t, _ := time.Parse(time.RFC3339, si.Session.CreatedAt)
	timeStr := t.Local().Format("2006-01-02 15:04")

	firstMsg := si.Session.FirstMsg
	if firstMsg == "" {
		firstMsg = "(empty session)"
	}
	firstMsg = strings.TrimSpace(firstMsg)
	firstMsg = strings.ReplaceAll(firstMsg, "\n", " ")

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


type Browser struct {
	list   list.Model
	store  SessionStore
	active bool
}

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

	perPage := height
	if perPage < 1 {
		perPage = 1
	}
	l.Paginator.PerPage = perPage

	return &Browser{list: l, store: store}
}


func (b *Browser) SetActive(active bool) {
	b.active = active
}

func (b *Browser) Active() bool {
	return b.active
}

func (b *Browser) IsEmpty() bool {
	return len(b.list.Items()) == 0
}


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

func (b *Browser) SetSessions(sessions []store.Session) {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{Session: s}
	}
	b.list.SetItems(items)
}

func (b *Browser) Selected() *store.Session {
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

func (b *Browser) GetMessages(sessionID string) ([]store.Message, error) {
	return b.store.GetSessionMessages(sessionID)
}


func (b *Browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newList, cmd := b.list.Update(msg)
	b.list = newList
	return nil, cmd
}

func (b *Browser) View() string {
	return b.list.View()
}

func (b *Browser) SetSize(width, height int) {
	b.list.SetSize(width, height)
}
