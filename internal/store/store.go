package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps an SQLite connection for session persistence.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at the given path.
// If path is empty, defaults to ~/.gocode/sessions.db.
// Automatically runs DDL migrations on open.
func Open(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("store: cannot determine home dir: %w", err)
		}
		path = filepath.Join(home, ".gocode", "sessions.db")
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("store: create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	// SQLite serializes writes; single connection is fine.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates tables and indexes if they don't exist.
func (s *Store) migrate() error {
	ddl := `
	CREATE TABLE IF NOT EXISTS sessions (
		id         TEXT PRIMARY KEY,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		model      TEXT NOT NULL,
		cwd        TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_created
		ON sessions(created_at DESC);
	CREATE TABLE IF NOT EXISTS messages (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id   TEXT NOT NULL,
		seq          INTEGER NOT NULL,
		msg_type     TEXT NOT NULL,
		content      TEXT NOT NULL DEFAULT '',
		tool_name    TEXT DEFAULT NULL,
		tool_args    TEXT DEFAULT NULL,
		tool_call_id TEXT DEFAULT NULL,
		has_error    INTEGER DEFAULT 0,
		created_at   TEXT NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_messages_session
		ON messages(session_id, seq);
	`
	_, err := s.db.Exec(ddl)
	return err
}
