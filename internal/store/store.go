package store

import (
	"bufio"
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
//
// Each session is stored in a single file under the store directory.
//   - Line 1:   JSON-encoded Session (compact, no newlines in values)
//   - Line 2..N: JSON-encoded Message, one per line
//
// File name: <SessionID>.session

// ---- Store ----

type Store struct {
	mu       sync.Mutex
	dir      string
	sessions map[string]*Session // SessionID → Session
	messages map[string][]Message // SessionID → ordered messages
}

// sessionFileName returns the file name for a given session ID.
func sessionFileName(id string) string {
	return id + ".session"
}

// defaultDir returns the default store directory (~/.gocode/sessions).
func defaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".gocode", "sessions")
	return dir
}

// Open opens (or creates) the session store at the given directory.
// If path is empty, uses ~/.gocode/sessions.
func Open(path string) (*Store, error) {
	if path == "" {
		path = defaultDir()
	}

	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}

	s := &Store{
		dir:      path,
		sessions: make(map[string]*Session),
		messages: make(map[string][]Message),
	}

	// Scan existing session files into memory.
	entries, err := os.ReadDir(path)
	if err != nil {
		return s, nil // empty store
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".session" {
			continue
		}
		s.loadSessionFile(filepath.Join(path, entry.Name()))
	}

	return s, nil
}

// loadSessionFile reads a single .session file into memory.
func (s *Store) loadSessionFile(filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Increase buffer for long lines (tool output may be large).
	scanner.Buffer(make([]byte, 0, 256*1024), 16*1024*1024)

	lineNum := 0
	var sessionID string
	for scanner.Scan() {
		line := scanner.Bytes()
		if lineNum == 0 {
			var sess Session
			if err := json.Unmarshal(line, &sess); err != nil {
				return // corrupted file, skip
			}
			s.sessions[sess.SessionID] = &sess
			sessionID = sess.SessionID
		} else {
			var msg Message
			if err := json.Unmarshal(line, &msg); err == nil {
				s.messages[sessionID] = append(s.messages[sessionID], msg)
			}
		}
		lineNum++
	}
}

// Close is a no-op; all data is written through to disk immediately.
func (s *Store) Close() error {
	return nil
}

// NewSessionID generates a new unique session identifier.
func NewSessionID() string {
	return uuid.New().String()
}

// EnsureSession creates a session record if it doesn't already exist.
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

	return s.writeSessionFile(id)
}

// ListSessions returns sessions ordered by CreatedAt descending.
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

// AppendMessage appends a message to the session and persists it to disk.
func (s *Store) AppendMessage(msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg.CreatedAt = time.Now().Format(time.RFC3339)

	s.messages[msg.SessionID] = append(s.messages[msg.SessionID], msg)

	// Update FirstMsg on the first user message of this session.
	needRewrite := false
	if sess, ok := s.sessions[msg.SessionID]; ok {
		if sess.FirstMsg == "" && msg.MsgType == "user" {
			sess.FirstMsg = msg.Content
			needRewrite = true
		}
	}

	// If the session header changed or the file doesn't exist yet,
	// rewrite the entire file. Otherwise just append the message line.
	filePath := filepath.Join(s.dir, sessionFileName(msg.SessionID))
	if needRewrite || !fileExists(filePath) {
		return s.writeSessionFile(msg.SessionID)
	}
	return s.appendMessageLine(filePath, msg)
}

// GetSessionMessages returns all persisted messages for a session.
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

// ---- internal helpers (caller must hold s.mu) ----

// writeSessionFile writes (or rewrites) the full session file:
// line 1 = Session JSON, remaining lines = Message JSON each.
func (s *Store) writeSessionFile(id string) error {
	sess, ok := s.sessions[id]
	if !ok {
		return nil
	}

	filePath := filepath.Join(s.dir, sessionFileName(id))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Line 1: Session JSON (compact, single line).
	sessJSON, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	f.Write(sessJSON)
	f.Write([]byte{'\n'})

	// Remaining lines: each message as a single JSON line.
	for _, msg := range s.messages[id] {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		f.Write(msgJSON)
		f.Write([]byte{'\n'})
	}

	return nil
}

// appendMessageLine appends a single message JSON line to an existing file.
func (s *Store) appendMessageLine(filePath string, msg Message) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = f.Write(append(msgJSON, '\n'))
	return err
}

// fileExists reports whether the given path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
