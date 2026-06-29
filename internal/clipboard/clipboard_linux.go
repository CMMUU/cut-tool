//go:build linux

// Linux adapter.
//
// Uses github.com/atotto/clipboard which detects the session (X11 vs
// Wayland) at init. Same poll-based Watch as the Windows adapter because
// there is no portable Linux change-notification API; wlr-data-control
// on Wayland and XFixes selection change on X11 would require separate
// code paths that are out of scope for this phase.
package clipboard

import (
	"context"
	"time"

	"github.com/atotto/clipboard"
)

type linuxClipboard struct {
	lastHash string
}

func newPlatformClipboard() Clipboard {
	return &linuxClipboard{lastHash: ""}
}

func (l *linuxClipboard) Read() (Entry, error) {
	s, err := clipboard.ReadAll()
	if err != nil {
		return Entry{}, err
	}
	if s == "" {
		return Entry{}, ErrEmpty
	}
	return Entry{Type: TypeText, Content: s, Preview: previewOf(s)}, nil
}

func (l *linuxClipboard) Write(e Entry) error {
	if e.Type != TypeText {
		return ErrUnsupportedType
	}
	if err := clipboard.WriteAll(e.Content); err != nil {
		return err
	}
	l.lastHash = hashOf(e.Content)
	return nil
}

func (l *linuxClipboard) Watch(ctx context.Context, out chan<- Entry) error {
	if s, err := clipboard.ReadAll(); err == nil && s != "" {
		l.lastHash = hashOf(s)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s, err := clipboard.ReadAll()
			if err != nil || s == "" {
				continue
			}
			h := hashOf(s)
			if h == l.lastHash {
				continue
			}
			l.lastHash = h
			select {
			case out <- Entry{Type: TypeText, Content: s, Preview: previewOf(s)}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
