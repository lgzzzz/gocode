package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lgzzzz/gocode/internal/agent"
)

// ---- types ----

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
	MsgType    string // uses agent.MsgXxx constants (MsgUser, MsgAssistant, etc.)
	Content    string
	ToolName   string // only for tool_call
	ToolArgs   string // only for tool_call
	ToolCallID string // shared by tool_call and tool_result
	HasError   bool   // tool_result or error
	CreatedAt  string // ISO 8601
}

// ---- session operations ----

// NewSessionID returns a new random UUID string for use as a session ID.
func NewSessionID() string {
	return uuid.New().String()
}

// EnsureSession creates a session row if it does not already exist.
// Safe to call multiple times – uses INSERT OR IGNORE.
func (s *Store) EnsureSession(id, model, cwd string) error {
	now := nowUTC()
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO sessions (id, created_at, updated_at, model, cwd)
		 VALUES (?, ?, ?, ?, ?)`,
		id, now, now, model, cwd,
	)
	if err != nil {
		return fmt.Errorf("store: ensure session: %w", err)
	}
	return nil
}

// ListSessions returns up to limit most recent sessions (by created_at DESC)
// with their message counts and the first user message.
func (s *Store) ListSessions(limit int) ([]SessionInfo, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.created_at, s.updated_at, s.model, s.cwd,
		        COUNT(m.id) AS msg_count,
		        (SELECT m2.content FROM messages m2
		         WHERE m2.session_id = s.id AND m2.msg_type = ?
		         ORDER BY m2.seq LIMIT 1) AS first_msg
		 FROM sessions s
		 LEFT JOIN messages m ON m.session_id = s.id
		 GROUP BY s.id
		 ORDER BY s.created_at DESC
		 LIMIT ?`,
		string(agent.MsgUser), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()

	var result []SessionInfo
	for rows.Next() {
		var si SessionInfo
		var firstMsg sql.NullString
		if err := rows.Scan(&si.ID, &si.CreatedAt, &si.UpdatedAt, &si.Model, &si.CWD, &si.MessageCount, &firstMsg); err != nil {
			return nil, fmt.Errorf("store: scan session: %w", err)
		}
		if firstMsg.Valid {
			si.FirstMsg = firstMsg.String
		}
		result = append(result, si)
	}
	return result, rows.Err()
}

// ---- message operations ----

// AppendMessage appends a complete message to the given session.
// seq is computed automatically (max + 1) inside a transaction, and
// the session's updated_at is refreshed.
func (s *Store) AppendMessage(msg Message) error {
	now := nowUTC()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Get current max seq for the session.
	var maxSeq sql.NullInt64
	err = tx.QueryRow(
		`SELECT MAX(seq) FROM messages WHERE session_id = ?`, msg.SessionID,
	).Scan(&maxSeq)
	if err != nil {
		return fmt.Errorf("store: query max seq: %w", err)
	}

	seq := 1
	if maxSeq.Valid {
		seq = int(maxSeq.Int64) + 1
	}

	// 2. Insert message.
	_, err = tx.Exec(
		`INSERT INTO messages (session_id, seq, msg_type, content,
			tool_name, tool_args, tool_call_id, has_error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.SessionID, seq, msg.MsgType, msg.Content,
		msg.ToolName, msg.ToolArgs, msg.ToolCallID,
		boolToInt(msg.HasError), now,
	)
	if err != nil {
		return fmt.Errorf("store: insert message: %w", err)
	}

	// 3. Update session timestamp.
	_, err = tx.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, now, msg.SessionID)
	if err != nil {
		return fmt.Errorf("store: update session time: %w", err)
	}

	return tx.Commit()
}

// GetSessionMessages returns all messages for the given session,
// ordered by seq ascending.
func (s *Store) GetSessionMessages(sessionID string) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT session_id, seq, msg_type, content,
		        tool_name, tool_args, tool_call_id, has_error, created_at
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY seq ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get session messages: %w", err)
	}
	defer rows.Close()

	var result []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.SessionID, &msg.Seq, &msg.MsgType, &msg.Content,
			&msg.ToolName, &msg.ToolArgs, &msg.ToolCallID, &msg.HasError, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan message: %w", err)
		}
		result = append(result, msg)
	}
	return result, rows.Err()
}

// ---- helpers ----

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
