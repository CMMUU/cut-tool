package clipboard

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"strings"

	_ "image/png" // register PNG decoder for image previews
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

// previewOfImage returns a one-line label for an image payload, e.g.
// "🖼 image 1200×800". It decodes only the header (DecodeConfig), so it is
// cheap even for large images; on failure it falls back to the byte size.
func previewOfImage(data []byte) string {
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		return fmt.Sprintf("🖼 image %d×%d", cfg.Width, cfg.Height)
	}
	return fmt.Sprintf("🖼 image (%d bytes)", len(data))
}

// hashOf returns a stable short hash used for de-duplication of identical
// consecutive clipboard captures (e.g. ctrl-C with nothing selected can
// fire a change event with the same content).
func hashOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}
