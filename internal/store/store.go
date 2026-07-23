package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session and Message definitions — DO NOT MODIFY (required by spec).

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

// ---- file format ----

type fileData struct {
	Sessions []Session `json:"sessions"`
	Messages []Message `json:"messages"`
}

// ---- Store ----

type Store struct {
	mu       sync.Mutex
	path     string
	sessions map[string]*Session // SessionID -> Session
	messages map[string][]Message // SessionID -> ordered messages
}

func defaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".gocode")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "sessions.json")
}

func Open(path string) (*Store, error) {
	if path == "" {
		path = defaultPath()
	}

	s := &Store{
		path:     path,
		sessions: make(map[string]*Session),
		messages: make(map[string][]Message),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil // fresh store
		}
		return nil, err
	}

	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		// If the file is corrupted, start fresh.
		return s, nil
	}

	for i := range fd.Sessions {
		sess := fd.Sessions[i]
		s.sessions[sess.SessionID] = &sess
	}
	for _, msg := range fd.Messages {
		s.messages[msg.SessionID] = append(s.messages[msg.SessionID], msg)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.flush()
}

func (s *Store) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var fd fileData
	for _, sess := range s.sessions {
		fd.Sessions = append(fd.Sessions, *sess)
	}
	// Sort sessions by CreatedAt descending for stable output
	sort.Slice(fd.Sessions, func(i, j int) bool {
		return fd.Sessions[i].CreatedAt > fd.Sessions[j].CreatedAt
	})

	for _, msgs := range s.messages {
		fd.Messages = append(fd.Messages, msgs...)
	}

	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func NewSessionID() string {
	return uuid.New().String()
}

func (s *Store) EnsureSession(id, model, cwd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; ok {
		return nil
	}

	s.sessions[id] = &Session{
		SessionID: id,
		CreatedAt: time.Now().Format(time.RFC3339),
		Model:     model,
		CWD:       cwd,
	}
	return s.flushLocked()
}

func (s *Store) ListSessions(limit int) ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		list = append(list, *sess)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt > list[j].CreatedAt
	})
	if limit > 0 && limit < len(list) {
		list = list[:limit]
	}
	return list, nil
}

func (s *Store) AppendMessage(msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg.CreatedAt = time.Now().Format(time.RFC3339)

	s.messages[msg.SessionID] = append(s.messages[msg.SessionID], msg)

	// Set FirstMsg on the first user message
	if sess, ok := s.sessions[msg.SessionID]; ok {
		if sess.FirstMsg == "" && msg.MsgType == "user" {
			sess.FirstMsg = msg.Content
		}
	}

	return s.flushLocked()
}

func (s *Store) GetSessionMessages(sessionID string) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgs := s.messages[sessionID]
	if msgs == nil {
		return []Message{}, nil
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

// flushLocked writes data to disk; caller must hold s.mu.
func (s *Store) flushLocked() error {
	var fd fileData
	for _, sess := range s.sessions {
		fd.Sessions = append(fd.Sessions, *sess)
	}
	sort.Slice(fd.Sessions, func(i, j int) bool {
		return fd.Sessions[i].CreatedAt > fd.Sessions[j].CreatedAt
	})

	for _, msgs := range s.messages {
		fd.Messages = append(fd.Messages, msgs...)
	}

	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}
