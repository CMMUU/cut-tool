package clipboard

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// previewOf returns a human-friendly one-line preview of a text payload.
// Long content is truncated with an ellipsis; newlines are flattened so
// the list row stays a single line.
func previewOf(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}

// hashOf returns a stable short hash used for de-duplication of identical
// consecutive clipboard captures (e.g. ctrl-C with nothing selected can
// fire a change event with the same content).
func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}
