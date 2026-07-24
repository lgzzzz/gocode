package store

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)


type Session struct {
	SessionID string
	CreatedAt string

	Model    string
	CWD      string
	FirstMsg string
}

type Message struct {
	SessionID string
	CreatedAt string

	MsgType string
	MsgID   string

	Content string

	ToolName   string
	ToolArgs   string
	ToolCallID string

	HasError bool
}



type Store struct {
	mu       sync.Mutex
	dir      string
	sessions map[string]*Session
	messages map[string][]Message
	writers  map[string]*os.File
}

func sessionFileName(id string) string {
	return id + ".session"
}

func defaultDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	dir := filepath.Join(cwd, ".gocode", "sessions")
	return dir
}

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

	entries, err := os.ReadDir(path)
	if err != nil {
		return s, nil
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

func (s *Store) loadSessionFile(filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 16*1024*1024)

	lineNum := 0
	var sessionID string
	for scanner.Scan() {
		line := scanner.Bytes()
		if lineNum == 0 {
			var sess Session
			if err := json.Unmarshal(line, &sess); err != nil {
				return
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

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, w := range s.writers {
		w.Close()
		delete(s.writers, id)
	}
	return nil
}

func NewSessionID() string {
	b := make([]byte, 3)
	rand.Read(b)
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b)
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

	return s.writeSessionFile(id)
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

	needRewrite := false
	if sess, ok := s.sessions[msg.SessionID]; ok {
		if sess.FirstMsg == "" && msg.MsgType == "user" {
			sess.FirstMsg = msg.Content
			needRewrite = true
		}
	}

	if needRewrite {
		return s.writeSessionFile(msg.SessionID)
	}

	return s.appendLine(msg.SessionID, msg)
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

func (s *Store) writeSessionFile(id string) error {
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

	sessJSON, err := json.Marshal(sess)
	if err != nil {
		f.Close()
		return err
	}
	f.Write(sessJSON)
	f.Write([]byte{'\n'})

	for _, msg := range s.messages[id] {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			f.Close()
			return err
		}
		f.Write(msgJSON)
		f.Write([]byte{'\n'})
	}

	f.Close()

	w, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	s.writers[id] = w
	return nil
}

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
