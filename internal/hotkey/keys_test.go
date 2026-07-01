package hotkey

import "testing"

func TestParseKey(t *testing.T) {
	// Known keys resolve; case-insensitive.
	if _, err := parseKey("v"); err != nil {
		t.Fatalf("parseKey(v): %v", err)
	}
	if _, err := parseKey("A"); err != nil {
		t.Fatalf("parseKey(A): %v", err)
	}
	// Empty falls back to the default key.
	if _, err := parseKey(""); err != nil {
		t.Fatalf("parseKey(empty): %v", err)
	}
	// Unknown key is rejected.
	if _, err := parseKey("F99"); err == nil {
		t.Fatal("parseKey(F99): expected error")
	}
}

func TestParseMods(t *testing.T) {
	// Empty falls back to platform defaults (non-empty).
	got, err := parseMods(nil)
	if err != nil {
		t.Fatalf("parseMods(nil): %v", err)
	}
	if len(got) == 0 {
		t.Fatal("parseMods(nil): expected platform default modifiers")
	}
	// Known names resolve.
	if _, err := parseMods([]string{"ctrl", "shift"}); err != nil {
		t.Fatalf("parseMods(ctrl,shift): %v", err)
	}
	// Unknown modifier is rejected.
	if _, err := parseMods([]string{"hyper"}); err == nil {
		t.Fatal("parseMods(hyper): expected error")
	}
}

func TestAvailableKeysSorted(t *testing.T) {
	if len(AvailableKeys) == 0 {
		t.Fatal("AvailableKeys must not be empty")
	}
	// First entry should be the letter A (rank 0).
	if AvailableKeys[0] != "A" {
		t.Fatalf("expected first key A, got %q", AvailableKeys[0])
	}
}
