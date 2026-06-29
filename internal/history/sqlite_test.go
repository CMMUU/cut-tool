package history

import (
	"context"
	"testing"
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

	e := FromClipboard("text", "hello world", "hello world")
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

	e := FromClipboard("text", "dup", "dup")
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
		_ = s.Append(ctx, FromClipboard("text", content, content))
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

	_ = s.Append(ctx, FromClipboard("text", "go is great", "go is great"))
	_ = s.Append(ctx, FromClipboard("text", "python rocks", "python rocks"))

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

	_ = s.Append(ctx, FromClipboard("text", "x", "x"))
	if err := s.Clear(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	recs, _ := s.List(ctx, 10)
	if len(recs) != 0 {
		t.Fatalf("expected empty after clear, got %d", len(recs))
	}
}
