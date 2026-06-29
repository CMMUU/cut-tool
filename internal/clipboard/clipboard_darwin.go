//go:build darwin

// Darwin adapter.
//
// Uses CGO + NSPasteboard. changeCount is the supported way to detect
// clipboard mutations; Apple's docs explicitly recommend polling it.
// A 300 ms tick is responsive enough for a paste-history UX and uses
// negligible CPU.
package clipboard

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation

#import <AppKit/AppKit.h>

static int pb_change_count(void) {
    return (int)[[NSPasteboard generalPasteboard] changeCount];
}

// Returns a heap-allocated C string the caller must free, or NULL when
// the clipboard has no string content.
static char* pb_read_string(void) {
    NSString *s = [[NSPasteboard generalPasteboard] stringForType:NSPasteboardTypeString];
    if (s == nil) return NULL;
    const char *utf8 = [s UTF8String];
    if (utf8 == NULL) return NULL;
    return strdup(utf8);
}

static void pb_write_string(const char *cstr) {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    [pb clearContents];
    NSString *s = [NSString stringWithUTF8String:cstr];
    [pb setString:s forType:NSPasteboardTypeString];
}

// Returns 1 if the pasteboard is marked as concealed by a password manager
// (org.nspasteboard.ConcealedType), 0 otherwise.
static int pb_is_concealed(void) {
    return [[NSPasteboard generalPasteboard].types
            containsObject:@"org.nspasteboard.ConcealedType"] ? 1 : 0;
}
*/
import "C"

import (
	"context"
	"errors"
	"time"
	"unsafe"
)

type darwinClipboard struct {
	lastChangeCount int
}

func newPlatformClipboard() Clipboard {
	return &darwinClipboard{lastChangeCount: -1}
}

func (d *darwinClipboard) Read() (Entry, error) {
	if C.pb_is_concealed() != 0 {
		return Entry{}, ErrEmpty
	}
	cstr := C.pb_read_string()
	if cstr == nil {
		return Entry{}, errors.New("darwin: clipboard has no string content")
	}
	defer C.free(unsafe.Pointer(cstr))

	content := C.GoString(cstr)
	return Entry{
		Type:    TypeText,
		Content: content,
		Preview: previewOf(content),
	}, nil
}

func (d *darwinClipboard) Write(e Entry) error {
	if e.Type != TypeText {
		return errors.New("darwin: only TypeText supported in this phase")
	}
	cs := C.CString(e.Content)
	defer C.free(unsafe.Pointer(cs))
	C.pb_write_string(cs)
	// Bump the cached count so our own writes don't re-trigger Watch.
	d.lastChangeCount = int(C.pb_change_count())
	return nil
}

func (d *darwinClipboard) Watch(ctx context.Context, out chan<- Entry) error {
	// Prime with the current count so we don't fire for state at startup.
	d.lastChangeCount = int(C.pb_change_count())

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			cc := int(C.pb_change_count())
			if cc == d.lastChangeCount {
				continue
			}
			d.lastChangeCount = cc
			// Skip entries marked as concealed by a password manager.
			if C.pb_is_concealed() != 0 {
				continue
			}
			e, err := d.Read()
			if err != nil {
				// Empty clipboard after a clear is a valid state — skip.
				continue
			}
			select {
			case out <- e:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
