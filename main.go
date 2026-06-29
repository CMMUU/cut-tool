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

	"cut-tool/internal/clipboard"
	"cut-tool/internal/history"
	"cut-tool/internal/hotkey"
	"cut-tool/internal/tray"
	"cut-tool/internal/ui"
)

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
				rec := history.FromClipboard(string(e.Type), e.Content, e.Preview)
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
		OnSelect: func(content string) {
			if werr := cb.Write(clipboard.Entry{
				Type:    clipboard.TypeText,
				Content: content,
				Preview: content,
			}); werr != nil {
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
					Preview:   r.Preview,
					Content:   r.Content,
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
	})

	// Attach system-tray menu (Fyne desktop.App, same main thread).
	tray.Setup(win.App(), tray.Options{
		OnShow: win.Show,
		OnClear: func() {
			if cerr := store.Clear(ctx); cerr != nil {
				log.Printf("clear history: %v", cerr)
			}
		},
		OnExit: cancel, // cancel ctx so goroutines exit cleanly
	})

	// Start global hotkey listener (Cmd+Shift+V / Ctrl+Shift+V).
	go func() {
		if err := hotkey.Register(ctx, win.Show); err != nil {
			log.Printf("hotkey: %v", err)
		}
	}()

	log.Println("cut-tool ready")
	win.Run() // blocks main thread until Quit
}
