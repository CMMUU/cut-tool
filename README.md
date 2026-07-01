# cut-tool

> Cross-platform clipboard history manager — lives in your system tray, stores every copy, lets you search and re-paste instantly.

[中文说明](#中文说明)

---

## Why cut-tool

Everyone uses a clipboard. Almost no one uses the same one twice.

macOS hides its history behind a setting most people never find. Windows has `Win+V`. Linux has a dozen tray applets, each with its own quirks. The moment you switch machines — or switch operating systems during a workday — the muscle memory breaks. Different shortcut, different window, different behavior, and your history stays trapped on the box that captured it.

cut-tool collapses that into one experience. The same tray icon, the same `Cmd/Ctrl+Shift+V`, the same search-and-repaste flow on all three platforms. Underneath, each OS is served by its native mechanism — CGO/NSPasteboard on macOS, `atotto/clipboard` on Windows and Linux — so capture is fast and correct everywhere; on top, the interaction is identical.

That consistency is a design choice, not an accident. History lives in a local SQLite database driven by pure-Go `modernc.org/sqlite` (no CGO), which is exactly what lets one codebase cross-compile and behave the same across macOS, Windows, and Linux. Nothing leaves your machine — no cloud, no sync, no account.

## Features

- **Clipboard history** — captures every text copy automatically
- **Instant search** — filter history with a real-time search box
- **Pin entries** — star (★) important snippets to keep them at the top
- **Global hotkey** — `Cmd+Shift+V` (macOS) / `Ctrl+Shift+V` (Windows/Linux) opens history from anywhere
- **Sensitive content filter** — skips password-manager entries (macOS: `org.nspasteboard.ConcealedType`)
- **System tray** — always running, zero taskbar footprint
- **Persistent storage** — SQLite, survives restarts, keeps the latest 500 entries

## Platform Support

| Platform | Clipboard | Hotkey | Tray |
|----------|-----------|--------|------|
| macOS 12+ | ✅ Native NSPasteboard | ✅ Cmd+Shift+V | ✅ |
| Windows 10+ | ✅ Win32 | ✅ Ctrl+Shift+V | ✅ |
| Linux (X11/Wayland) | ✅ xclip/xsel | ✅ Ctrl+Shift+V | ⚠️ Requires tray-capable DE |

## Installation

### Build from source

**macOS**
```bash
# Requires Xcode Command Line Tools
xcode-select --install
go build -o cut-tool .
./cut-tool
```

**Linux**
```bash
sudo apt install gcc libgl1-mesa-dev xorg-dev
go build -o cut-tool .
./cut-tool
```

**Windows** (requires [MinGW-w64](https://www.mingw-w64.org/))
```cmd
go build -o cut-tool.exe .
cut-tool.exe
```

### macOS permissions

The global hotkey requires **Accessibility** permission:
> System Settings → Privacy & Security → Accessibility → add the `go` binary or the built app

## Usage

1. Launch `cut-tool` — a ✂ icon appears in the menu bar / system tray
2. Copy text as usual — it's captured automatically
3. Press `Cmd+Shift+V` (or click **Show History** in the tray) to open the history window
4. Type to search, click an entry to paste it back to the clipboard
5. Click ☆ to pin an entry — pinned items always appear at the top

## Architecture

```
systray (tray) ──► ui (Fyne window)
      │                   │
      ▼                   ▼
         history (SQLite)
      ▲                   ▲
         clipboard (platform adapters)
               │
          OS Pasteboard
```

See [`SDD.md`](SDD.md) for the full software design document.

## Development

```bash
go test ./...          # run all tests
go build ./...         # verify build
go run .               # run locally
```

## License

MIT

---

## 中文说明

**cut-tool** 是一个跨平台桌面剪贴板增强工具，常驻系统托盘，自动记录每次复制内容，支持快速搜索和一键回填。

### 为什么做这个

剪贴板人人在用，但几乎没人用的是同一套。

macOS 的历史藏在多数人从没打开过的设置里，Windows 靠 `Win+V`，Linux 则有一堆各具脾气的托盘小工具。一旦换台机器——或者一个工作日里在几个系统间来回切——肌肉记忆就断了：快捷键不同、窗口不同、行为不同，历史还各存各的，锁死在当初捕获它的那台设备上。

cut-tool 把这些收敛成同一种体验：三端都是同一个托盘图标、同一个 `Cmd/Ctrl+Shift+V`、同一套搜索与回填流程。底层各走原生机制——macOS 用 CGO/NSPasteboard，Windows 和 Linux 用 `atotto/clipboard`——保证各平台捕获都快而准；上层交互则完全一致。

这种一致性是刻意的工程取舍，而非巧合。历史记录存在本地 SQLite，由纯 Go 的 `modernc.org/sqlite` 驱动（无 CGO），正是这一点让同一份代码能交叉编译、在 macOS / Windows / Linux 上表现如一。数据不出本机——无云端、无同步、无账号。

### 功能

- **剪贴板历史** — 自动捕获每次复制的文字
- **实时搜索** — 在搜索框输入关键字即时过滤
- **置顶收藏** — 点击 ★ 固定常用内容，始终排在最前
- **全局快捷键** — `Cmd+Shift+V`（macOS）/ `Ctrl+Shift+V`（Win/Linux）随时唤出
- **敏感内容过滤** — 自动跳过密码管理器写入的条目（macOS）
- **SQLite 持久化** — 重启不丢失，保留最近 500 条

### 快速开始

```bash
# macOS
go build -o cut-tool . && ./cut-tool

# Linux
sudo apt install gcc libgl1-mesa-dev xorg-dev
go build -o cut-tool . && ./cut-tool
```

首次运行需在 **系统设置 → 隐私与安全性 → 辅助功能** 中为 `go` 授权，之后全局快捷键才能生效。

详细设计见 [`SDD.md`](SDD.md)。
