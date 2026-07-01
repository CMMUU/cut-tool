package clipboard

import "testing"

func TestPreviewOf(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"  trim  ", "trim"},
		{"a\nb\tc", "a b c"},
		{string(make([]byte, 100)), string(make([]byte, 80)) + "..."},
	}
	for _, c := range cases {
		if got := previewOf(c.in); got != c.want {
			t.Errorf("previewOf(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHashOf(t *testing.T) {
	if hashOf([]byte("")) == hashOf([]byte("x")) {
		t.Fatal("different inputs must produce different hashes")
	}
	if hashOf([]byte("abc")) != hashOf([]byte("abc")) {
		t.Fatal("same input must produce same hash")
	}
}
