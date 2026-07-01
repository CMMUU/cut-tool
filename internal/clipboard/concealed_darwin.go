//go:build darwin

package clipboard

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation

#import <AppKit/AppKit.h>

// Returns 1 if the pasteboard is marked as concealed by a password manager
// (org.nspasteboard.ConcealedType), 0 otherwise.
static int pb_is_concealed(void) {
    return [[NSPasteboard generalPasteboard].types
            containsObject:@"org.nspasteboard.ConcealedType"] ? 1 : 0;
}
*/
import "C"

// isConcealed reports whether the current macOS pasteboard is flagged as
// sensitive by a password manager, so we can skip capturing it.
func isConcealed() bool {
	return C.pb_is_concealed() != 0
}
