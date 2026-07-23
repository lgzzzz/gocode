package store

import (
	"github.com/google/uuid"
)

// -4--- types ----

type SessionInfo struct {
	ID        string
	CreatedAt string
	UpdatedAt string
	Model     string
	CWD       string
	FirstMsg  string // first user message, for list display
}

// Message represents a single persisted message.
type Message struct {
	SessionID string

	MsgType string
	MsgID   string

	Content string

	ToolName   string // only for tool_call
	ToolArgs   string // only for tool_call
	ToolCallID string // shared by tool_call and tool_result

	HasError bool // tool_result or error
}

type Store struct {
}

func Open(path string) (*Store, error) {
	panic("not implement")
}

// Close is a no-op for file-based storage.
func (s *Store) Close() error {
	panic("not implement")
}

func NewSessionID() string {
	return uuid.New().String()
}

func (s *Store) EnsureSession(id, model, cwd string) error {
	panic("not implement")
}

func (s *Store) ListSessions(limit int) ([]SessionInfo, error) {
	panic("not implement")
}

func (s *Store) AppendMessage(msg Message) error {
	panic("not implement")
}
func (s *Store) GetSessionMessages(sessionID string) ([]Message, error) {
	panic("not implement")
}
