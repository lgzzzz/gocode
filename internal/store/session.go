package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
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
}

// Message represents a single persisted message.
type Message struct {
	SessionID  string
	Seq        int    // assigned automatically by AppendMessage
	Role       string // "user" | "assistant" | "system" | "tool"
	MsgType    string // see msg_type enum in docs
	Content    string
	ToolName   string // only for tool_call
	ToolArgs   string // only for tool_call
	ToolCallID string // shared by tool_call and tool_result
	HasError   bool   // tool_result or error
	CreatedAt  string // ISO 8601
}

// ---- session operations ----

// CreateSession inserts a new session and returns its UUID.
func (s *Store) CreateSession(model, cwd string) (string, error) {
	id := uuid.New().String()
	now := nowUTC()
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, created_at, updated_at, model, cwd)
		 VALUES (?, ?, ?, ?, ?)`,
		id, now, now, model, cwd,
	)
	if err != nil {
		return "", fmt.Errorf("store: create session: %w", err)
	}
	return id, nil
}

// UpdateSessionTime updates the updated_at field for the given session.
func (s *Store) UpdateSessionTime(sessionID string) {
	_, _ = s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, nowUTC(), sessionID)
}

// ListSessions returns up to limit most recent sessions (by created_at DESC)
// with their message counts.
func (s *Store) ListSessions(limit int) ([]SessionInfo, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.created_at, s.updated_at, s.model, s.cwd,
		        COUNT(m.id) AS msg_count
		 FROM sessions s
		 LEFT JOIN messages m ON m.session_id = s.id
		 GROUP BY s.id
		 ORDER BY s.created_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()

	var result []SessionInfo
	for rows.Next() {
		var si SessionInfo
		if err := rows.Scan(&si.ID, &si.CreatedAt, &si.UpdatedAt, &si.Model, &si.CWD, &si.MessageCount); err != nil {
			return nil, fmt.Errorf("store: scan session: %w", err)
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
		`INSERT INTO messages (session_id, seq, role, msg_type, content,
			tool_name, tool_args, tool_call_id, has_error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.SessionID, seq, msg.Role, msg.MsgType, msg.Content,
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
