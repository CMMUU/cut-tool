// Package history persists clipboard entries to a local SQLite database and
// provides search/dedupe/cap operations.
package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// Store is the persistence interface used by the rest of the app.
// Defined as an interface so the SQLite impl can be swapped for an in-memory
// fake in tests.
type Store interface {
	// Append inserts a new entry or, if an entry with the same hash already
	// exists, refreshes its created_at so it bubbles to the top of the list.
	Append(ctx context.Context, e Entry) error

	// List returns up to `limit` most-recent entries, newest first.
	List(ctx context.Context, limit int) ([]Record, error)

	// Search returns up to `limit` entries whose content or preview contains
	// query (case-insensitive substring match via SQL LIKE).
	Search(ctx context.Context, query string, limit int) ([]Record, error)

	// SetPinned marks or unmarks an entry as pinned. Pinned entries appear
	// at the top of List results regardless of created_at.
	SetPinned(ctx context.Context, hash string, pinned bool) error

	// Clear deletes every entry.
	Clear(ctx context.Context) error

	// Close releases the underlying database handle.
	Close() error
}

// Entry is what the caller hands to Append. The clipboard package's Entry
// type is structurally compatible and can be converted with FromClipboard.
type Entry struct {
	Type    string
	Content string
	Preview string
	Hash    string
}

// FromClipboard builds a history Entry from the raw fields of a
// clipboard.Entry, computing the dedup hash from the content. It takes the
// fields rather than the clipboard.Entry type itself to avoid an import
// cycle between the two packages.
func FromClipboard(typ, content, preview string) Entry {
	return Entry{
		Type:    typ,
		Content: content,
		Preview: preview,
		Hash:    hashOf(content),
	}
}

// hashOf returns a stable short hash used as the dedup key. It mirrors the
// clipboard package's hashOf so identical content produces the same key
// regardless of which package computed it.
func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// Record is one row returned by List/Search.
type Record struct {
	ID        int64
	Type      string
	Content   string
	Preview   string
	Hash      string
	Pinned    bool
	CreatedAt time.Time
}

// ErrNotFound is returned when no record matches an operation. (Currently
// unused but reserved for future single-record lookups.)
var ErrNotFound = errors.New("history: not found")
