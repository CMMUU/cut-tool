// Package main is the entry point of cut-tool.
//
// Architecture:
//
//	+-------------------+    +---------------------+
//	|  systray (tray)   |--->|   ui (Fyne window)  |
//	+-------------------+    +---------------------+
//	         |                          |
//	         v                          v
//	+-------------------------------------------+
//	|             history (SQLite)              |
//	+-------------------------------------------+
//	         ^                          ^
//	         |                          |
//	+-------------------------------------------+
//	|        clipboard (platform adapter)      |
//	+-------------------------------------------+
//	         ^
//	         |
//	   OS Pasteboard
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fyne.io/fyne/v2"

	"cut-tool/internal/clipboard"
	"cut-tool/internal/config"
	"cut-tool/internal/history"
	"cut-tool/internal/hotkey"
	"cut-tool/internal/tray"
	"cut-tool/internal/ui"
)

// maxImageBytes caps the size of a single captured image (10 MiB). Larger
// images are skipped so the history database stays bounded.
const maxImageBytes = 10 << 20

// cleanupInterval is how often the background pass deletes expired unpinned
// entries. Cleanup also runs once at startup.
const cleanupInterval = 6 * time.Hour

// go run . (base) ➜  cut-tool git:(main) ✗ go run .
// # cut-tool
// ld: warning: ignoring duplicate libraries: '-lobjc'
// 2026/06/29 19:14:41 main.go:39: cut-tool starting...
// 2026/06/29 19:14:42 main.go:144: cut-tool ready
// 2026/06/29 19:14:42 main.go:140: hotkey: hotkey: failed to register, grant the application Accessibility (Input Monitoring) permission
// 2026/06/29 19:14:52 main.go:85: captured: (base) ➜  cut-tool git:(main) ✗ go run . # cut-tool ld: warning: ignoring du...
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("cut-tool starting...")

	// Root context — canceled on SIGINT/SIGTERM or when Fyne quits.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Catch terminal signals (e.g. kill from a shell).
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	// Open history store.
	store, err := history.SQLiteOpen(ctx)
	if err != nil {
		log.Fatalf("open history store: %v", err)
	}
	defer func() {
		if cerr := store.Close(); cerr != nil {
			log.Printf("close history store: %v", cerr)
		}
	}()

	// Load user settings (retention period, etc.).
	cfg, err := config.Open()
	if err != nil {
		log.Printf("config: %v; using defaults", err)
	}

	// Cleanup loop: delete expired unpinned entries at startup and periodically.
	go runCleanup(ctx, store, cfg)

	// Declared here so the UI's SetHotkey callback can capture it; assigned
	// after the window is built (the manager needs win.Show).
	var hkMgr *hotkey.Manager

	// Start clipboard watcher in background.
	cb := clipboard.New()
	entries := make(chan clipboard.Entry, 16)
	go func() {
		if werr := cb.Watch(ctx, entries); werr != nil {
			log.Printf("clipboard watch stopped: %v", werr)
		}
	}()

	// Capture loop: persist every new clipboard entry.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-entries:
				// Skip oversized images to keep the DB bounded.
				if e.Type == clipboard.TypeImage && len(e.Data) > maxImageBytes {
					log.Printf("skip image: %d bytes exceeds limit", len(e.Data))
					continue
				}
				rec := history.FromClipboard(string(e.Type), e.Content, e.Data, e.Preview)
				if aerr := store.Append(ctx, rec); aerr != nil {
					log.Printf("append entry: %v", aerr)
					continue
				}
				log.Printf("captured: %s", e.Preview)
			}
		}
	}()

	// Build UI (does not show the window yet).
	win := ui.NewWindow(ui.Deps{
		OnSelect: func(item ui.Item) {
			e := clipboard.Entry{Type: clipboard.EntryType(item.Type)}
			switch item.Type {
			case string(clipboard.TypeImage):
				e.Data = item.Data
				e.Preview = item.Preview
			default:
				e.Type = clipboard.TypeText
				e.Content = item.Content
				e.Preview = item.Content
			}
			if werr := cb.Write(e); werr != nil {
				log.Printf("write clipboard: %v", werr)
			}
		},
		OnSearch: func(query string) []ui.Item {
			recs, err := store.Search(ctx, query, 100)
			if err != nil {
				log.Printf("search: %v", err)
				return nil
			}
			items := make([]ui.Item, len(recs))
			for i, r := range recs {
				items[i] = ui.Item{
					Type:      r.Type,
					Preview:   r.Preview,
					Content:   r.Content,
					Data:      r.Data,
					Hash:      r.Hash,
					Pinned:    r.Pinned,
					Timestamp: r.CreatedAt.Format("01-02 15:04"),
				}
			}
			return items
		},
		OnPin: func(hash string, pinned bool) {
			if err := store.SetPinned(ctx, hash, pinned); err != nil {
				log.Printf("set pinned: %v", err)
			}
		},
		OnDelete: func(hash string) {
			if err := store.Delete(ctx, hash); err != nil {
				log.Printf("delete entry: %v", err)
			}
		},
		OnClear: func() {
			if err := store.Clear(ctx); err != nil {
				log.Printf("clear history: %v", err)
			}
		},
		GetRetentionDays: func() int {
			if cfg == nil {
				return config.DefaultRetentionDays
			}
			return cfg.Get().RetentionDays
		},
		SetRetentionDays: func(days int) error {
			if cfg == nil {
				return nil
			}
			if err := cfg.SetRetentionDays(days); err != nil {
				return err
			}
			// Apply immediately so the new window reflects the change.
			runCleanupOnce(ctx, store, cfg)
			return nil
		},
		AvailableMods: hotkey.AvailableMods,
		AvailableKeys: hotkey.AvailableKeys,
		GetHotkey: func() ([]string, string) {
			if cfg == nil {
				return hotkey.DefaultModNames, hotkey.DefaultKeyName
			}
			c := cfg.Get()
			mods := c.HotkeyMods
			if len(mods) == 0 {
				mods = hotkey.DefaultModNames
			}
			key := c.HotkeyKey
			if key == "" {
				key = hotkey.DefaultKeyName
			}
			return mods, key
		},
		SetHotkey: func(mods []string, key string) error {
			if hkMgr == nil {
				return nil
			}
			// Persist first, then (re)register on a separate goroutine.
			// Registering must NOT run inline here: this callback fires from
			// within Fyne's glfw event-poll callback, and the macOS Carbon
			// RegisterEventHotKey CGO call traps if invoked from that context.
			if cfg != nil {
				if err := cfg.SetHotkey(mods, key); err != nil {
					return err
				}
			}
			go func() {
				if err := hkMgr.Apply(mods, key); err != nil {
					log.Printf("hotkey: apply %v+%s: %v", mods, key, err)
				}
			}()
			return nil
		},
		DefaultRetentionDays: config.DefaultRetentionDays,
		DefaultHotkeyMods:    hotkey.DefaultModNames,
		DefaultHotkeyKey:     hotkey.DefaultKeyName,
	})

	// showWindow marshals win.Show onto Fyne's main thread. The hotkey
	// listener and tray callbacks fire from other goroutines, and Fyne 2.7
	// requires UI mutations to run via fyne.Do.
	showWindow := func() { fyne.Do(win.Show) }

	// Attach system-tray menu (Fyne desktop.App, same main thread).
	tray.Setup(win.App(), tray.Options{
		OnShow:     showWindow,
		OnSettings: func() { fyne.Do(win.ShowSettings) },
		OnClear: func() {
			if cerr := store.Clear(ctx); cerr != nil {
				log.Printf("clear history: %v", cerr)
			}
		},
		OnExit: cancel, // cancel ctx so goroutines exit cleanly
	})

	// Start global hotkey listener, using the configured combination (falls
	// back to platform defaults when unset).
	hkMgr = hotkey.NewManager(ctx, showWindow)
	go func() {
		mods, key := hotkey.DefaultModNames, hotkey.DefaultKeyName
		if cfg != nil {
			c := cfg.Get()
			if len(c.HotkeyMods) > 0 {
				mods = c.HotkeyMods
			}
			if c.HotkeyKey != "" {
				key = c.HotkeyKey
			}
		}
		if err := hkMgr.Apply(mods, key); err != nil {
			log.Printf("hotkey: %v", err)
		}
	}()

	log.Println("cut-tool ready")
	win.Run() // blocks main thread until Quit
}

// runCleanup deletes expired unpinned entries at startup and then every
// cleanupInterval until ctx is canceled.
func runCleanup(ctx context.Context, store history.Store, cfg *config.Store) {
	runCleanupOnce(ctx, store, cfg)
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCleanupOnce(ctx, store, cfg)
		}
	}
}

// runCleanupOnce performs a single expiry pass using the current retention
// setting. Pinned entries are never removed (enforced in the store).
func runCleanupOnce(ctx context.Context, store history.Store, cfg *config.Store) {
	days := config.DefaultRetentionDays
	if cfg != nil {
		days = cfg.Get().RetentionDays
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	n, err := store.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		log.Printf("cleanup: %v", err)
		return
	}
	if n > 0 {
		log.Printf("cleanup: removed %d expired entries (older than %d days)", n, days)
	}
}
