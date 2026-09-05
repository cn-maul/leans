package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite connection and exposes settings/history persistence.
// Subjects live in the subject package and are loaded from markdown files,
// so this type no longer owns them.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database and initializes tables.
// dbDir is the directory where the .db file should live; if empty it
// defaults to "./data" relative to the working directory.
func NewStore(dbDir string) (*Store, error) {
	if dbDir == "" {
		dbDir = "data"
	}

	dbPath := filepath.Join(dbDir, "leans.db")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// modernc.org/sqlite is a pure-Go single-process driver; a single
	// connection avoids SQLITE_BUSY contention in this small app.
	db.SetMaxOpenConns(1)

	if err := initTables(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func initTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS history (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			subject        TEXT NOT NULL,
			question       TEXT NOT NULL,
			category       TEXT NOT NULL DEFAULT '',
			result         TEXT NOT NULL DEFAULT '',
			model          TEXT NOT NULL DEFAULT '',
			tokens         INTEGER NOT NULL DEFAULT 0,
			elapsed_ms     INTEGER NOT NULL DEFAULT 0,
			first_token_ms INTEGER NOT NULL DEFAULT 0,
			created_at     TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS questions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			subject     TEXT NOT NULL,
			stem        TEXT NOT NULL,
			options     TEXT NOT NULL DEFAULT '[]',
			answer      TEXT NOT NULL DEFAULT '',
			category    TEXT NOT NULL DEFAULT '',
			sub_category TEXT NOT NULL DEFAULT '',
			source      TEXT NOT NULL DEFAULT '',
			year        TEXT NOT NULL DEFAULT '',
			difficulty  TEXT NOT NULL DEFAULT '',
			tags        TEXT NOT NULL DEFAULT '[]',
			notes       TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS knowledge_units (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			subject        TEXT NOT NULL,
			title          TEXT NOT NULL,
			section        TEXT NOT NULL DEFAULT '',
			content        TEXT NOT NULL DEFAULT '',
			kind           TEXT NOT NULL DEFAULT '',
			keywords       TEXT NOT NULL DEFAULT '[]',
			rule           TEXT NOT NULL DEFAULT '',
			example        TEXT NOT NULL DEFAULT '',
			trap           TEXT NOT NULL DEFAULT '',
			source_section TEXT NOT NULL DEFAULT '',
			created_at     TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS study_records (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			subject       TEXT NOT NULL,
			question_id   INTEGER NOT NULL DEFAULT 0,
			history_id    INTEGER NOT NULL DEFAULT 0,
			user_answer   TEXT NOT NULL DEFAULT '',
			correct       INTEGER NOT NULL DEFAULT 0,
			difficulty    TEXT NOT NULL DEFAULT '',
			review_at     TEXT NOT NULL DEFAULT '',
			skill_change  TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("init table: %w", err)
		}
	}
	return nil
}

// migrate applies additive schema changes for databases created by older
// versions. New installs get the full schema from initTables.
func migrate(db *sql.DB) error {
	if err := ensureColumn(db, "history", "result", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return fmt.Errorf("migrate history.result: %w", err)
	}
	if err := ensureColumn(db, "history", "tokens", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("migrate history.tokens: %w", err)
	}
	if err := ensureColumn(db, "history", "elapsed_ms", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("migrate history.elapsed_ms: %w", err)
	}
	if err := ensureColumn(db, "history", "model", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return fmt.Errorf("migrate history.model: %w", err)
	}
	if err := ensureColumn(db, "history", "first_token_ms", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("migrate history.first_token_ms: %w", err)
	}
	return nil
}

// ensureColumn adds a column when it is missing from the table.
func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}
