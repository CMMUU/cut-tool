// Unified cross-platform clipboard adapter built on
// golang.design/x/clipboard, which supports both UTF-8 text and PNG image
// data on macOS, Windows and Linux.
//
// The library exposes a change-notification Watch per format, so we run one
// watcher for text and one for images and merge them onto a single out
// channel. De-duplication uses a content hash so our own writes (and
// duplicate change events that fire for a single copy touching multiple
// formats) do not re-enter the history.
//
// Platform note: on macOS and Linux this pulls in CGO/X11 at build time
// (see go.mod). Windows stays pure Go. Password-manager "concealed" content
// is filtered on macOS via the small CGO helper in concealed_darwin.go.
package clipboard

import (
	"bytes"
	"context"
	"sync"

	"golang.design/x/clipboard"
)

type sysClipboard struct {
	initOnce sync.Once
	initErr  error

	mu       sync.Mutex
	lastHash string // hash of the last entry we saw or wrote, for dedupe
}

func newPlatformClipboard() Clipboard {
	return &sysClipboard{}
}

// ensureInit lazily initializes the clipboard library. On Linux this fails
// when no X11 display is available; callers surface the error rather than
// panicking (the library's own Init panics on failure).
func (c *sysClipboard) ensureInit() error {
	c.initOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				c.initErr = errInit(r)
			}
		}()
		c.initErr = clipboard.Init()
	})
	return c.initErr
}

func (c *sysClipboard) setLastHash(h string) {
	c.mu.Lock()
	c.lastHash = h
	c.mu.Unlock()
}

func (c *sysClipboard) Read() (Entry, error) {
	if err := c.ensureInit(); err != nil {
		return Entry{}, err
	}
	if isConcealed() {
		return Entry{}, ErrEmpty
	}
	if img := clipboard.Read(clipboard.FmtImage); len(img) > 0 {
		return imageEntry(img), nil
	}
	if txt := clipboard.Read(clipboard.FmtText); len(txt) > 0 {
		s := string(txt)
		return Entry{Type: TypeText, Content: s, Preview: previewOf(s)}, nil
	}
	return Entry{}, ErrEmpty
}

func (c *sysClipboard) Write(e Entry) error {
	if err := c.ensureInit(); err != nil {
		return err
	}
	switch e.Type {
	case TypeText:
		clipboard.Write(clipboard.FmtText, []byte(e.Content))
		c.setLastHash(hashOf([]byte(e.Content)))
	case TypeImage:
		if len(e.Data) == 0 {
			return ErrEmpty
		}
		clipboard.Write(clipboard.FmtImage, e.Data)
		c.setLastHash(hashOf(e.Data))
	default:
		return ErrUnsupportedType
	}
	return nil
}

func (c *sysClipboard) Watch(ctx context.Context, out chan<- Entry) error {
	if err := c.ensureInit(); err != nil {
		return err
	}

	// Prime lastHash with the current content so we don't emit startup state.
	if img := clipboard.Read(clipboard.FmtImage); len(img) > 0 {
		c.setLastHash(hashOf(img))
	} else if txt := clipboard.Read(clipboard.FmtText); len(txt) > 0 {
		c.setLastHash(hashOf(txt))
	}

	// A single Watch observes both formats; each Data is tagged with its format.
	ch := clipboard.Watch(ctx, clipboard.FmtText, clipboard.FmtImage)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-ch:
			if !ok {
				return nil
			}
			if len(d.Bytes) == 0 {
				continue
			}
			var e Entry
			switch d.Format {
			case clipboard.FmtImage:
				e = imageEntry(d.Bytes)
			case clipboard.FmtText:
				s := string(d.Bytes)
				e = Entry{Type: TypeText, Content: s, Preview: previewOf(s)}
			default:
				continue
			}
			c.emit(ctx, out, e, d.Bytes)
		}
	}
}

// emit filters concealed content, de-dupes by hash, and forwards e to out.
func (c *sysClipboard) emit(ctx context.Context, out chan<- Entry, e Entry, raw []byte) {
	if len(raw) == 0 {
		return
	}
	if isConcealed() {
		return
	}
	h := hashOf(raw)
	c.mu.Lock()
	dup := h == c.lastHash
	c.lastHash = h
	c.mu.Unlock()
	if dup {
		return
	}
	select {
	case out <- e:
	case <-ctx.Done():
	}
}

// imageEntry builds a TypeImage entry, defensively copying the library's
// buffer (Watch reuses it across events).
func imageEntry(b []byte) Entry {
	data := bytes.Clone(b)
	return Entry{Type: TypeImage, Data: data, Preview: previewOfImage(data)}
}
