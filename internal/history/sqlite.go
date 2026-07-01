package history

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// Pure-Go SQLite driver — avoids CGO toolchain.
	_ "modernc.org/sqlite"
)

// DefaultMaxEntries caps how many rows the store keeps. Older rows are
// trimmed by Append once we exceed the cap.
const DefaultMaxEntries = 500

// dbFileName is the SQLite database file inside the per-app config dir.
const dbFileName = "history.db"

// SQLiteOpen opens (creating if necessary) the history database under the
// user's config directory and applies the schema. The returned Store must
// be Close()d on shutdown.
func SQLiteOpen(ctx context.Context) (Store, error) {
	return SQLiteOpenWithCap(ctx, DefaultMaxEntries)
}

// SQLiteOpenWithCap is SQLiteOpen with a custom capacity override. Mostly
// useful for tests; production code should call SQLiteOpen.
func SQLiteOpenWithCap(ctx context.Context, cap int) (Store, error) {
	if cap <= 0 {
		cap = DefaultMaxEntries
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("history: locate user config dir: %w", err)
	}
	dir := filepath.Join(cfgDir, "cut-tool")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("history: mkdir %s: %w", dir, err)
	}
	return SQLiteOpenAt(ctx, filepath.Join(dir, dbFileName), cap)
}

// SQLiteOpenAt opens the database at an explicit path. Useful in tests to
// avoid touching the real user config directory.
func SQLiteOpenAt(ctx context.Context, dbPath string, cap int) (Store, error) {
	if cap <= 0 {
		cap = DefaultMaxEntries
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("history: open sqlite at %s: %w", dbPath, err)
	}

	const schema = `
		CREATE TABLE IF NOT EXISTS entries (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			type        TEXT    NOT NULL,
			content     TEXT    NOT NULL,
			preview     TEXT    NOT NULL,
			hash        TEXT    NOT NULL UNIQUE,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_entries_created_at ON entries(created_at DESC);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("history: apply schema: %w", err)
	}
	// Migration: add pinned column to existing databases (ignored if present).
	_, _ = db.ExecContext(ctx, `ALTER TABLE entries ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`)
	// Migration: add data column for binary payloads such as images.
	_, _ = db.ExecContext(ctx, `ALTER TABLE entries ADD COLUMN data BLOB`)

	return &sqliteStore{db: db, cap: cap}, nil
}

type sqliteStore struct {
	db  *sql.DB
	cap int
}

func (s *sqliteStore) Close() error { return s.db.Close() }

func (s *sqliteStore) Append(ctx context.Context, e Entry) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("history: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Upsert: insert or refresh timestamp on hash collision.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO entries (type, content, data, preview, hash, created_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(hash) DO UPDATE SET created_at = CURRENT_TIMESTAMP`,
		e.Type, e.Content, e.Data, e.Preview, e.Hash,
	); err != nil {
		return fmt.Errorf("history: upsert: %w", err)
	}

	// Trim oldest rows beyond cap. id DESC is the tiebreaker for entries
	// sharing the same created_at second (common in fast loops / tests).
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM entries
		 WHERE id IN (
		   SELECT id FROM entries ORDER BY created_at DESC, id DESC LIMIT -1 OFFSET ?
		 )`, s.cap,
	); err != nil {
		return fmt.Errorf("history: trim: %w", err)
	}

	return tx.Commit()
}

func (s *sqliteStore) List(ctx context.Context, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, content, data, preview, hash, pinned, created_at
		 FROM entries ORDER BY pinned DESC, created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("history: list: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *sqliteStore) Search(ctx context.Context, query string, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 100
	}
	pat := "%" + escapeLike(query) + "%"
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, content, data, preview, hash, pinned, created_at
		 FROM entries
		 WHERE content LIKE ? ESCAPE '\' OR preview LIKE ? ESCAPE '\'
		 ORDER BY pinned DESC, created_at DESC, id DESC LIMIT ?`,
		pat, pat, limit)
	if err != nil {
		return nil, fmt.Errorf("history: search: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *sqliteStore) SetPinned(ctx context.Context, hash string, pinned bool) error {
	v := 0
	if pinned {
		v = 1
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE entries SET pinned = ? WHERE hash = ?`, v, hash,
	); err != nil {
		return fmt.Errorf("history: set pinned: %w", err)
	}
	return nil
}

// DeleteOlderThan removes unpinned entries created before cutoff. Pinned
// entries are kept regardless of age.
func (s *sqliteStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	// SQLite stores created_at as UTC text ("2006-01-02 15:04:05"); compare
	// against the same format so the string comparison is chronological.
	cut := cutoff.UTC().Format("2006-01-02 15:04:05")
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM entries WHERE pinned = 0 AND created_at < ?`, cut)
	if err != nil {
		return 0, fmt.Errorf("history: delete older than: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *sqliteStore) Delete(ctx context.Context, hash string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM entries WHERE hash = ?`, hash); err != nil {
		return fmt.Errorf("history: delete: %w", err)
	}
	return nil
}

func (s *sqliteStore) Clear(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM entries`); err != nil {
		return fmt.Errorf("history: clear: %w", err)
	}
	return nil
}

// escapeLike escapes LIKE wildcards in user input so a search for "100%"
// does not match everything.
func escapeLike(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '%' || c == '_' || c == '\\' {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return string(out)
}

func scanRows(rows *sql.Rows) ([]Record, error) {
	out := make([]Record, 0)
	for rows.Next() {
		var (
			r          Record
			data       []byte
			pinned     int
			createdRaw string
		)
		if err := rows.Scan(&r.ID, &r.Type, &r.Content, &data, &r.Preview, &r.Hash, &pinned, &createdRaw); err != nil {
			return nil, fmt.Errorf("history: scan: %w", err)
		}
		r.Data = data
		r.Pinned = pinned != 0
		r.CreatedAt = parseSQLiteTime(createdRaw)
		out = append(out, r)
	}
	return out, rows.Err()
}

// parseSQLiteTime accepts the formats SQLite commonly emits and falls back
// to time.Now on failure rather than failing the whole query.
func parseSQLiteTime(s string) time.Time {
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Now()
}
