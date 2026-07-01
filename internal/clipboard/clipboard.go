// Package clipboard provides a cross-platform clipboard abstraction.
//
// Each platform (darwin, windows, linux) has its own adapter implementing
// the Clipboard interface. Selection happens via build tags (files named
// clipboard_<goos>.go).
package clipboard

import "context"

// Entry represents a single clipboard capture.
//
// For TypeText the payload is Content (UTF-8) and Data is nil. For TypeImage
// the payload is Data (PNG bytes) and Content is empty. Preview is always a
// short human-readable label for the history list.
type Entry struct {
	Type    EntryType
	Content string // text payload (TypeText)
	Data    []byte // binary payload, e.g. PNG bytes (TypeImage)
	Preview string // short text shown in the history list
}

// EntryType is the discriminator for Entry.
type EntryType string

const (
	TypeText  EntryType = "text"
	TypeImage EntryType = "image"
	// TypeHTML, TypeFile reserved for future phases.
)

// Clipboard is the cross-platform interface every adapter must implement.
//
// Watch streams every *new* clipboard entry to out. It returns when ctx is
// canceled. Adapters should debounce: clipboard change events can fire
// multiple times for a single user action (e.g. a copy that touches
// several formats), and the caller relies on a single Entry per user action.
type Clipboard interface {
	// Read returns the current clipboard content. Returns an error if the
	// clipboard is empty or holds an unsupported type.
	Read() (Entry, error)

	// Write sets the clipboard content.
	Write(Entry) error

	// Watch streams new entries to out until ctx is canceled.
	Watch(ctx context.Context, out chan<- Entry) error
}

// New returns the clipboard adapter for the current platform. The concrete
// implementation is selected at compile time via build tags (see
// clipboard_<goos>.go).
func New() Clipboard {
	return newPlatformClipboard()
}
