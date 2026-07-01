//go:build !darwin

package clipboard

// isConcealed is a no-op on platforms without a password-manager
// concealed-content convention. Windows and Linux have no portable
// equivalent of macOS's org.nspasteboard.ConcealedType.
func isConcealed() bool { return false }
