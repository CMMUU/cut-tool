package history

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func openTestStore(t *testing.T) Store {
	t.Helper()
	path := t.TempDir() + "/test.db"
	s, err := SQLiteOpenAt(context.Background(), path, 5)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAppendAndList(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	e := FromClipboard("text", "hello world", nil, "hello world")
	if err := s.Append(ctx, e); err != nil {
		t.Fatalf("append: %v", err)
	}

	recs, err := s.List(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(recs) != 1 || recs[0].Content != "hello world" {
		t.Fatalf("unexpected records: %+v", recs)
	}
}

func TestDedup(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	e := FromClipboard("text", "dup", nil, "dup")
	_ = s.Append(ctx, e)
	_ = s.Append(ctx, e) // same hash → upsert, not new row

	recs, _ := s.List(ctx, 10)
	if len(recs) != 1 {
		t.Fatalf("expected 1 record after dedup, got %d", len(recs))
	}
}

func TestCapTrim(t *testing.T) {
	s := openTestStore(t) // cap=5
	ctx := context.Background()

	for i := range 7 {
		content := string(rune('a' + i)) // a, b, c, …
		_ = s.Append(ctx, FromClipboard("text", content, nil, content))
	}

	recs, _ := s.List(ctx, 10)
	if len(recs) != 5 {
		t.Fatalf("expected 5 records after trim, got %d", len(recs))
	}
	// Newest entry ("g") must be first.
	if recs[0].Content != "g" {
		t.Fatalf("expected newest first, got %q", recs[0].Content)
	}
}

func TestSearch(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.Append(ctx, FromClipboard("text", "go is great", nil, "go is great"))
	_ = s.Append(ctx, FromClipboard("text", "python rocks", nil, "python rocks"))

	recs, err := s.Search(ctx, "go", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(recs) != 1 || recs[0].Content != "go is great" {
		t.Fatalf("unexpected search result: %+v", recs)
	}
}

func TestClear(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.Append(ctx, FromClipboard("text", "x", nil, "x"))
	if err := s.Clear(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	recs, _ := s.List(ctx, 10)
	if len(recs) != 0 {
		t.Fatalf("expected empty after clear, got %d", len(recs))
	}
}

func TestImageRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x01, 0x02, 0x03}
	e := FromClipboard("image", "", png, "🖼 image 2×2")
	if err := s.Append(ctx, e); err != nil {
		t.Fatalf("append image: %v", err)
	}

	recs, err := s.List(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	r := recs[0]
	if r.Type != "image" {
		t.Fatalf("expected type image, got %q", r.Type)
	}
	if !bytes.Equal(r.Data, png) {
		t.Fatalf("image bytes round-tripped incorrectly: got %v", r.Data)
	}
	if r.Content != "" {
		t.Fatalf("expected empty content for image, got %q", r.Content)
	}
}

func TestImageDedupByBytes(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	png := []byte{0x89, 'P', 'N', 'G', 0x01, 0x02}
	e := FromClipboard("image", "", png, "🖼 image")
	_ = s.Append(ctx, e)
	_ = s.Append(ctx, e) // same bytes → same hash → upsert, not a new row

	recs, _ := s.List(ctx, 10)
	if len(recs) != 1 {
		t.Fatalf("expected 1 record after image dedup, got %d", len(recs))
	}
}

func TestDeleteOlderThan(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	// Insert two entries, then backdate one of them well past the cutoff.
	_ = s.Append(ctx, FromClipboard("text", "old", nil, "old"))
	_ = s.Append(ctx, FromClipboard("text", "fresh", nil, "fresh"))
	store := s.(*sqliteStore)
	oldTime := time.Now().AddDate(0, 0, -30).UTC().Format("2006-01-02 15:04:05")
	if _, err := store.db.ExecContext(ctx,
		`UPDATE entries SET created_at = ? WHERE content = 'old'`, oldTime); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	cutoff := time.Now().AddDate(0, 0, -15)
	n, err := s.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("delete older than: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 deleted, got %d", n)
	}
	recs, _ := s.List(ctx, 10)
	if len(recs) != 1 || recs[0].Content != "fresh" {
		t.Fatalf("expected only 'fresh' to remain, got %+v", recs)
	}
}

func TestDeleteOlderThanKeepsPinned(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	e := FromClipboard("text", "keep", nil, "keep")
	_ = s.Append(ctx, e)
	_ = s.SetPinned(ctx, e.Hash, true)
	store := s.(*sqliteStore)
	oldTime := time.Now().AddDate(0, 0, -30).UTC().Format("2006-01-02 15:04:05")
	_, _ = store.db.ExecContext(ctx,
		`UPDATE entries SET created_at = ? WHERE content = 'keep'`, oldTime)

	n, err := s.DeleteOlderThan(ctx, time.Now().AddDate(0, 0, -15))
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 deleted (pinned kept), got %d", n)
	}
	recs, _ := s.List(ctx, 10)
	if len(recs) != 1 {
		t.Fatalf("pinned entry must survive cleanup, got %d rows", len(recs))
	}
}
