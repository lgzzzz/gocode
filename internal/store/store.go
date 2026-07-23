package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// -4--- types ----

// SessionInfo holds the metadata for a single session.
type SessionInfo struct {
	ID           string
	CreatedAt    string
	UpdatedAt    string
	Model        string
	CWD          string
	MessageCount int
	FirstMsg     string // first user message, for list display
}

// Message represents a single persisted message.
type Message struct {
	SessionID  string
	Seq        int    // assigned automatically by AppendMessage
	MsgType    string // uses agent.MsgXxx constants
	Content    string
	Reasoning  string // reasoning_content (DeepSeek), for assistant messages
	ToolName   string // only for tool_call
	ToolArgs   string // only for tool_call
	ToolCallID string // shared by tool_call and tool_result
	HasError   bool   // tool_result or error
	CreatedAt  string // ISO 8601
}

// Store manages session persistence using JSONL files.
// Layout:
//
//	~/.gocode/sessions/
//	  <cwd-sanitized>/
//	    <session-id>.jsonl
//	    <session-id>.jsonl
//	  <cwd-sanitized>/
//	    ...
//
// Each session file contains one JSON object per line (JSONL format):
//   - Line 1: session metadata (id, created_at, updated_at, model, cwd)
//   - Line 2+: messages
//
// The file's modification time is used as updated_at for efficient listing.
type Store struct {
	baseDir string // ~/.gocode/sessions

	mu      sync.RWMutex
	sessDir map[string]string // sessionID -> cwdDir (in-memory cache)
}

// Open opens (or creates) the sessions directory at the given path.
// If path is empty, defaults to ~/.gocode/sessions.
func Open(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("store: cannot determine home dir: %w", err)
		}
		path = filepath.Join(home, ".gocode", "sessions")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("store: create sessions dir: %w", err)
	}

	return &Store{
		baseDir: path,
		sessDir: make(map[string]string),
	}, nil
}

// Close is a no-op for file-based storage.
func (s *Store) Close() error {
	return nil
}

// ---- helpers ----

// cwdDir returns a sanitized directory name for a given working directory path.
func cwdDir(cwd string) string {
	// Normalize separators and replace problematic chars.
	cwd = filepath.ToSlash(cwd)
	cwd = strings.TrimPrefix(cwd, "/")
	cleaned := strings.NewReplacer(
		":", "_",
		"/", "_",
		"\\", "_",
	).Replace(cwd)

	// Collapse consecutive underscores.
	for strings.Contains(cleaned, "__") {
		cleaned = strings.ReplaceAll(cleaned, "__", "_")
	}
	cleaned = strings.Trim(cleaned, "_")

	if cleaned == "" {
		cleaned = "root"
	}
	return cleaned
}

// sessionFilePath returns the full path to a session's JSONL file.
func (s *Store) sessionFilePath(cwd, sessionID string) string {
	return filepath.Join(s.baseDir, cwdDir(cwd), sessionID+".jsonl")
}

// findSessionFile searches all cwd directories for a session file by ID.
// Returns the full path and the cwd directory name, or empty strings if not found.
func (s *Store) findSessionFile(sessionID string) (filePath, cwdDirName string) {
	// Check in-memory cache first.
	s.mu.RLock()
	cached, ok := s.sessDir[sessionID]
	s.mu.RUnlock()
	if ok {
		p := filepath.Join(s.baseDir, cached, sessionID+".jsonl")
		if _, err := os.Stat(p); err == nil {
			return p, cached
		}
		// Cache is stale — fall through to scan.
	}

	// Scan all cwd directories.
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(s.baseDir, e.Name(), sessionID+".jsonl")
		if _, err := os.Stat(p); err == nil {
			// Update cache.
			s.mu.Lock()
			s.sessDir[sessionID] = e.Name()
			s.mu.Unlock()
			return p, e.Name()
		}
	}
	return "", ""
}

// ---- session metadata (line 1 of each file) ----

type sessionMeta struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Model     string `json:"model"`
	CWD       string `json:"cwd"`
}

// ---- session operations ----

// NewSessionID returns a new random UUID string for use as a session ID.
func NewSessionID() string {
	return uuid.New().String()
}

// EnsureSession creates a session file if it does not already exist.
func (s *Store) EnsureSession(id, model, cwd string) error {
	dir := filepath.Join(s.baseDir, cwdDir(cwd))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("store: create session dir: %w", err)
	}

	fpath := filepath.Join(dir, id+".jsonl")

	// If file already exists, do nothing.
	if _, err := os.Stat(fpath); err == nil {
		// Update cache.
		s.mu.Lock()
		s.sessDir[id] = cwdDir(cwd)
		s.mu.Unlock()
		return nil
	}

	now := nowUTC()
	meta := sessionMeta{
		Type:      "session",
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Model:     model,
		CWD:       cwd,
	}

	f, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("store: create session file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(meta); err != nil {
		return fmt.Errorf("store: write session meta: %w", err)
	}

	// Update cache.
	s.mu.Lock()
	s.sessDir[id] = cwdDir(cwd)
	s.mu.Unlock()

	return nil
}

// ListSessions returns up to limit most recent sessions across all cwd directories,
// sorted by updated_at (file modification time) descending.
func (s *Store) ListSessions(limit int) ([]SessionInfo, error) {
	var sessions []SessionInfo

	// Walk all cwd directories.
	cwdEntries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: read sessions dir: %w", err)
	}

	for _, cwdEntry := range cwdEntries {
		if !cwdEntry.IsDir() {
			continue
		}
		cwdDirPath := filepath.Join(s.baseDir, cwdEntry.Name())

		fileEntries, err := os.ReadDir(cwdDirPath)
		if err != nil {
			continue
		}

		for _, fe := range fileEntries {
			if fe.IsDir() || !strings.HasSuffix(fe.Name(), ".jsonl") {
				continue
			}

			fpath := filepath.Join(cwdDirPath, fe.Name())
			info, err := fe.Info()
			if err != nil {
				continue
			}

			// Read first line (metadata) to get full info.
			meta, firstMsg, msgCount, err := readSessionMeta(fpath)
			if err != nil || meta == nil {
				continue
			}

			si := SessionInfo{
				ID:           meta.ID,
				CreatedAt:    meta.CreatedAt,
				UpdatedAt:    info.ModTime().UTC().Format(time.RFC3339),
				Model:        meta.Model,
				CWD:          meta.CWD,
				MessageCount: msgCount,
				FirstMsg:     firstMsg,
			}
			sessions = append(sessions, si)
		}
	}

	// Sort by updated_at descending (use file mod time).
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})

	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

// readSessionMeta reads the first line (session metadata) and optionally
// the first user message from a session file. Returns message count as well.
func readSessionMeta(fpath string) (*sessionMeta, string, int, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, "", 0, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var meta sessionMeta
	if err := dec.Decode(&meta); err != nil {
		return nil, "", 0, err
	}

	if meta.Type != "session" {
		return nil, "", 0, fmt.Errorf("invalid session file: first line is not metadata")
	}

	// Scan remaining lines for first user message and message count.
	firstUserMsg := ""
	msgCount := 0
	for dec.More() {
		var msg map[string]any
		if err := dec.Decode(&msg); err != nil {
			break
		}
		msgCount++
		if firstUserMsg == "" {
			if t, ok := msg["type"].(string); ok && t == "user" {
				if c, ok := msg["content"].(string); ok {
					firstUserMsg = c
				}
			}
		}
	}

	return &meta, firstUserMsg, msgCount, nil
}

// ---- message operations ----

// AppendMessage appends a message line to the session's JSONL file.
// The Seq field is computed automatically (ignoring the incoming value).
func (s *Store) AppendMessage(msg Message) error {
	fpath, _ := s.findSessionFile(msg.SessionID)
	if fpath == "" {
		return fmt.Errorf("store: session %s not found", msg.SessionID)
	}

	// Compute next seq by counting existing message lines (skip meta line).
	nextSeq := 1 + countMessageLines(fpath)

	now := nowUTC()
	msg.CreatedAt = now
	msg.Seq = nextSeq

	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("store: open session file for append: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(messageToJSON(msg)); err != nil {
		return fmt.Errorf("store: append message: %w", err)
	}

	return nil
}

// countMessageLines counts the number of message lines (excluding meta line 1)
// in a session file. Returns 0 on error.
func countMessageLines(fpath string) int {
	f, err := os.Open(fpath)
	if err != nil {
		return 0
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	// Skip meta line.
	var meta map[string]any
	if err := dec.Decode(&meta); err != nil {
		return 0
	}

	count := 0
	for dec.More() {
		var raw map[string]any
		if err := dec.Decode(&raw); err != nil {
			break
		}
		count++
	}
	return count
}

// messageToJSON converts a Message to a map for JSON encoding.
func messageToJSON(msg Message) map[string]any {
	m := map[string]any{
		"type":       msg.MsgType,
		"seq":        msg.Seq,
		"content":    msg.Content,
		"created_at": msg.CreatedAt,
	}
	if msg.Reasoning != "" {
		m["reasoning"] = msg.Reasoning
	}
	if msg.ToolName != "" {
		m["tool_name"] = msg.ToolName
	}
	if msg.ToolArgs != "" {
		m["tool_args"] = msg.ToolArgs
	}
	if msg.ToolCallID != "" {
		m["tool_call_id"] = msg.ToolCallID
	}
	if msg.HasError {
		m["has_error"] = true
	}
	return m
}

// GetSessionMessages returns all messages for the given session,
// ordered by line order (which is insertion order).
func (s *Store) GetSessionMessages(sessionID string) ([]Message, error) {
	fpath, _ := s.findSessionFile(sessionID)
	if fpath == "" {
		return nil, fmt.Errorf("store: session %s not found", sessionID)
	}

	f, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("store: open session file: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	// Skip first line (metadata).
	var meta map[string]any
	if err := dec.Decode(&meta); err != nil {
		return nil, fmt.Errorf("store: read session meta: %w", err)
	}

	var result []Message
	seq := 0
	for dec.More() {
		var raw map[string]any
		if err := dec.Decode(&raw); err != nil {
			break
		}
		seq++

		msg := Message{
			SessionID:  sessionID,
			Seq:        seq,
			MsgType:    stringField(raw, "type"),
			Content:    stringField(raw, "content"),
			Reasoning:  stringField(raw, "reasoning"),
			ToolName:   stringField(raw, "tool_name"),
			ToolArgs:   stringField(raw, "tool_args"),
			ToolCallID: stringField(raw, "tool_call_id"),
			HasError:   boolField(raw, "has_error"),
			CreatedAt:  stringField(raw, "created_at"),
		}
		result = append(result, msg)
	}
	return result, nil
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func boolField(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// ---- helpers ----

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
