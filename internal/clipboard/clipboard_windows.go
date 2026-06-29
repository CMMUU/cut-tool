//go:build windows

// Windows adapter.
//
// Uses github.com/atotto/clipboard for read/write. For Watch we poll at
// 500 ms intervals and compare a SHA-256 prefix to dedupe identical
// captures. A future phase can swap polling for AddClipboardFormatListener
// (HWND message WM_CLIPBOARDUPDATE) to be truly event-driven.
package clipboard

import (
	"context"
	"time"

	"github.com/atotto/clipboard"
)

type windowsClipboard struct {
	lastHash string
}

func newPlatformClipboard() Clipboard {
	return &windowsClipboard{lastHash: ""}
}

func (w *windowsClipboard) Read() (Entry, error) {
	s, err := clipboard.ReadAll()
	if err != nil {
		return Entry{}, err
	}
	if s == "" {
		return Entry{}, ErrEmpty
	}
	return Entry{Type: TypeText, Content: s, Preview: previewOf(s)}, nil
}

func (w *windowsClipboard) Write(e Entry) error {
	if e.Type != TypeText {
		return ErrUnsupportedType
	}
	if err := clipboard.WriteAll(e.Content); err != nil {
		return err
	}
	// Update lastHash so our own write doesn't immediately fire Watch.
	w.lastHash = hashOf(e.Content)
	return nil
}

func (w *windowsClipboard) Watch(ctx context.Context, out chan<- Entry) error {
	// Prime.
	if s, err := clipboard.ReadAll(); err == nil && s != "" {
		w.lastHash = hashOf(s)
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
			if h == w.lastHash {
				continue
			}
			w.lastHash = h
			select {
			case out <- Entry{Type: TypeText, Content: s, Preview: previewOf(s)}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
