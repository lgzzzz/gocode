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
	sessions map[string]*Session  // SessionID -> Session
	messages map[string][]Message // SessionID -> ordered messages
	writers  map[string]*os.File  // SessionID -> open file handle (append mode)
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
		writers:  make(map[string]*os.File),
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
	sort.Slice(s.messages[sessionID], func(i, j int) bool {
		return s.messages[sessionID][i].CreatedAt < s.messages[sessionID][j].CreatedAt
	})
}

// Close closes all open file handles. Call on program exit (Ctrl+C).
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, w := range s.writers {
		w.Close()
		delete(s.writers, id)
	}
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

	// If FirstMsg changed (first user message), rewrite the entire file
	// so the session header on line 1 stays correct.
	if needRewrite {
		return s.writeSessionFile(msg.SessionID)
	}

	// Fast path: append a single line using the open file handle.
	return s.appendLine(msg.SessionID, msg)
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

// getOrOpenWriter returns the cached file handle for the session,
// or opens one in append mode if none is cached yet.
func (s *Store) getOrOpenWriter(id string) (*os.File, error) {
	if w, ok := s.writers[id]; ok {
		return w, nil
	}
	filePath := filepath.Join(s.dir, sessionFileName(id))
	w, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	s.writers[id] = w
	return w, nil
}

// writeSessionFile writes (or rewrites) the full session file.
// Closes the previous handle (if any), creates a fresh file, writes
// all data, then opens a new handle for future appends.
func (s *Store) writeSessionFile(id string) error {
	// Close and remove old cached handle.
	if old, ok := s.writers[id]; ok {
		old.Close()
		delete(s.writers, id)
	}

	sess, ok := s.sessions[id]
	if !ok {
		return nil
	}

	filePath := filepath.Join(s.dir, sessionFileName(id))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	// Line 1: Session JSON (compact, single line).
	sessJSON, err := json.Marshal(sess)
	if err != nil {
		f.Close()
		return err
	}
	f.Write(sessJSON)
	f.Write([]byte{'\n'})

	// Remaining lines: each message as a single JSON line.
	for _, msg := range s.messages[id] {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			f.Close()
			return err
		}
		f.Write(msgJSON)
		f.Write([]byte{'\n'})
	}

	// Keep the file open for future appends; swap the handle.
	f.Close()

	// Reopen in append mode and cache.
	w, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	s.writers[id] = w
	return nil
}

// appendLine appends a single message JSON line using the cached
// file handle. Opens the handle lazily if needed.
func (s *Store) appendLine(sessionID string, msg Message) error {
	w, err := s.getOrOpenWriter(sessionID)
	if err != nil {
		return err
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(append(msgJSON, '\n'))
	return err
}
