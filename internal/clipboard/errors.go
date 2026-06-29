package clipboard

import "errors"

// ErrEmpty is returned by Read when the clipboard holds no text content.
// Callers should treat this as a benign state (the user cleared the
// clipboard or the last copy produced no text).
var ErrEmpty = errors.New("clipboard: empty")

// ErrUnsupportedType is returned by Write when the caller asks for a
// content type the adapter does not implement yet.
var ErrUnsupportedType = errors.New("clipboard: only TypeText supported in this phase")
