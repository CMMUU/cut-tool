package config

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s, err := OpenAt(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if got := s.Get().RetentionDays; got != DefaultRetentionDays {
		t.Fatalf("expected default %d, got %d", DefaultRetentionDays, got)
	}
}

func TestSetAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s, err := OpenAt(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.SetRetentionDays(30); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Reopen from disk; the value must persist.
	s2, err := OpenAt(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := s2.Get().RetentionDays; got != 30 {
		t.Fatalf("expected persisted 30, got %d", got)
	}
}

func TestNormalizeClampsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s, _ := OpenAt(path)
	if err := s.SetRetentionDays(0); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := s.Get().RetentionDays; got != DefaultRetentionDays {
		t.Fatalf("expected clamp to %d, got %d", DefaultRetentionDays, got)
	}
}
