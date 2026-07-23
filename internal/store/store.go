package store

import (
	"github.com/google/uuid"
)

// todo C:\Users\LGZ\IdeaProjects\gocode\internal\store\store.go 需要在这个文件中实现会话持久化, 要求Session和Message的定义绝对不能更改, 除了修改本文件,不要修改任何其他的文件

type Session struct {
	SessionID string
	CreatedAt string

	Model    string
	CWD      string
	FirstMsg string // first user message, for list display
}

// Message represents a single persisted message.
type Message struct {
	SessionID string
	CreatedAt string

	MsgType string
	MsgID   string

	Content string // shared by tool_result, thinking message, assistant message, user message

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

func (s *Store) ListSessions(limit int) ([]Session, error) {
	panic("not implement")
}

func (s *Store) AppendMessage(msg Message) error {
	panic("not implement")
}
func (s *Store) GetSessionMessages(sessionID string) ([]Message, error) {
	panic("not implement")
}
