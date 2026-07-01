package clipboard

import (
	"errors"
	"fmt"
)

// ErrEmpty is returned by Read when the clipboard holds no supported content.
// Callers should treat this as a benign state (the user cleared the
// clipboard or the last copy produced no text/image).
var ErrEmpty = errors.New("clipboard: empty")

// ErrUnsupportedType is returned by Write when the caller asks for a
// content type the adapter does not implement yet.
var ErrUnsupportedType = errors.New("clipboard: unsupported content type")

// errInit wraps a panic recovered from clipboard.Init (the library panics
// instead of returning an error when the platform backend is unavailable,
// e.g. no X11 display on Linux).
func errInit(r any) error {
	return fmt.Errorf("clipboard: init failed: %v", r)
}
