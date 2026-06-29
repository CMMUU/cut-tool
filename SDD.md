# cut-tool 软件设计文档（SDD）

| 项 | 内容 |
| --- | --- |
| 文档名称 | cut-tool 软件设计规范文档（Software Design Document） |
| 项目代号 | cut-tool |
| 版本 | v0.1（Phase 1 骨架阶段） |
| 文档状态 | 草稿 |
| 最后更新 | 2026-06-29 |
| 维护者 | CangLu |
| 适用范围 | macOS / Windows / Linux 桌面端 |

---

## 1. 概述

### 1.1 项目定位

cut-tool 是一个**跨平台桌面剪贴板增强工具**。它常驻系统托盘，
后台监听系统剪贴板的变化，把每一次复制内容持久化为可检索的历史记录，
用户可随时调出窗口浏览、搜索历史条目，并一键将任意历史条目重新写回剪贴板。

### 1.2 目标

- 后台静默运行，对系统资源占用极低（轮询而非常驻 GUI 渲染）。
- 历史记录本地持久化，重启不丢失。
- 提供快速搜索与一键回填剪贴板的交互。
- 一套代码覆盖 macOS / Windows / Linux 三端。

### 1.3 非目标（当前阶段不做）

- 图片 / 文件 / 富文本（HTML）类型的剪贴板内容（仅预留类型枚举）。
- 云端同步、多设备共享。
- 端到端加密存储。
- 插件体系 / 脚本扩展。

---

## 2. 术语表

| 术语 | 含义 |
| --- | --- |
| Entry | 一次剪贴板捕获的内容单元（类型 + 原始内容 + 预览）。 |
| Record | 历史库中的一行持久化记录（Entry + 主键 + 哈希 + 时间戳）。 |
| Watch | 后台监听剪贴板变化并向通道推送新 Entry 的长循环。 |
| changeCount | macOS NSPasteboard 提供的变更计数器，用于检测剪贴板是否被修改。 |
| dedupe | 去重：对内容哈希相同的连续捕获只保留一条。 |
| cap | 历史库容量上限，超出后按时间淘汰最旧记录。 |

---

## 3. 系统架构

### 3.1 分层架构图

```
        +-------------------+      +---------------------+
        |  systray (tray)   |----->|   ui (Fyne window)  |
        |  托盘菜单/入口     |      |  搜索框 + 历史列表   |
        +-------------------+      +---------------------+
                 |                            |
                 | OnShow/OnClear/OnExit      | OnSelect/OnSearch
                 v                            v
        +-------------------------------------------+
        |             history (SQLite)              |
        |   Append / List / Search / Clear / Close  |
        +-------------------------------------------+
                 ^                            ^
                 | Append(new Entry)          | List/Search 回填
                 |                            |
        +-------------------------------------------+
        |        clipboard (平台适配层)             |
        |     Read / Write / Watch(ctx, out)        |
        +-------------------------------------------+
                 ^
                 | build tag 选择 darwin/windows/linux
                 v
                OS Pasteboard（系统剪贴板）
```

### 3.2 模块职责

| 模块 | 路径 | 职责 | 当前状态 |
| --- | --- | --- | --- |
| main | `main.go` | 进程入口，组装各模块并启动主循环。 | stub |
| clipboard | `internal/clipboard/` | 跨平台剪贴板读/写/监听抽象。 | 三端实现已完成 |
| history | `internal/history/` | 历史记录持久化（SQLite）、搜索、去重、容量淘汰。 | 实现已完成（有冲突文件待清理） |
| tray | `internal/tray/` | 系统托盘图标与菜单，应用主入口。 | stub |
| ui | `internal/ui/` | Fyne 主窗口：搜索框 + 历史列表 + 回填。 | stub |

### 3.3 关键数据流

**写入流（捕获）**：
```
OS 复制操作 → clipboard.Watch 检测变化 → 推送 Entry 到 out 通道
            → main 消费通道 → history.Append（去重 + 容量淘汰）→ SQLite
```

**读取流（回填）**：
```
用户在 UI 选中条目 → ui.OnSelect(content) → clipboard.Write(Entry)
                  → 系统剪贴板更新（同时刷新 lastHash/changeCount 避免自触发）
```

---

## 4. 核心数据模型

### 4.1 clipboard.Entry（运行时内存模型）

来源：`internal/clipboard/clipboard.go`

```go
type Entry struct {
    Type    EntryType // 内容类型，当前仅 TypeText
    Content string    // 原始内容（文本为 UTF-8）
    Preview string    // 历史列表展示用的单行短预览
}

type EntryType string
const TypeText EntryType = "text" // TypeImage/TypeHTML/TypeFile 预留
```

### 4.2 history 持久化模型

来源：`internal/history/store.go`

```go
type Entry struct {  // 调用方传入 Append
    Type, Content, Preview, Hash string
}

type Record struct { // List/Search 返回
    ID        int64
    Type      string
    Content   string
    Preview   string
    Hash      string
    CreatedAt time.Time
}
```

### 4.3 SQLite Schema

来源：`internal/history/sqlite.go`

```sql
CREATE TABLE IF NOT EXISTS entries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    type        TEXT     NOT NULL,
    content     TEXT     NOT NULL,
    preview     TEXT     NOT NULL,
    hash        TEXT     NOT NULL UNIQUE,   -- 去重键
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_entries_created_at ON entries(created_at DESC);
```

- **存储位置**：`os.UserConfigDir()/cut-tool/history.db`
- **并发模式**：WAL（`journal_mode(WAL)`），开启外键约束。
- **去重策略**：`hash` 唯一约束 + `ON CONFLICT(hash) DO UPDATE SET created_at = CURRENT_TIMESTAMP`，即重复内容不新增行，而是刷新时间戳令其置顶。
- **容量淘汰**：每次 Append 后执行 `DELETE ... ORDER BY created_at DESC LIMIT -1 OFFSET <cap>`，保留最新 `cap` 条（默认 500）。

---

## 5. 模块详细设计

### 5.1 clipboard 包（平台适配层）

**统一接口**（`clipboard.go`）：

```go
type Clipboard interface {
    Read() (Entry, error)                          // 读当前剪贴板
    Write(Entry) error                             // 写入剪贴板
    Watch(ctx context.Context, out chan<- Entry) error // 监听变化直到 ctx 取消
}
```

**平台选择机制**：通过 Go build tag（`//go:build darwin|windows|linux`）在编译期选定实现，
三端各自暴露内部构造函数 `newPlatformClipboard() Clipboard`。

| 平台 | 文件 | 底层机制 | 监听方式 | 轮询间隔 |
| --- | --- | --- | --- | --- |
| macOS | `clipboard_darwin.go` | CGO + NSPasteboard（AppKit） | 轮询 `changeCount` | 300ms |
| Windows | `clipboard_windows.go` | `atotto/clipboard` | 轮询 + SHA-256 前缀比对去重 | 500ms |
| Linux | `clipboard_linux.go` | `atotto/clipboard`（X11/Wayland 自动探测） | 轮询 + 哈希比对 | 500ms |

**设计要点**：
- **自写防抖**：`Write` 后立即更新缓存的 `lastHash`（Win/Linux）或 `lastChangeCount`（macOS），
  避免应用自己的写回操作被 Watch 当成一次新捕获重新入库。
- **启动预热（prime）**：Watch 开始时先记录当前状态，避免把启动瞬间的剪贴板内容误当成新捕获。
- **辅助函数**（`helpers.go`）：`previewOf` 生成单行 80 字截断预览；`hashOf` 生成 SHA-256 前 8 字节去重哈希。
- **错误语义**（`errors.go`）：`ErrEmpty`（空剪贴板，良性）、`ErrUnsupportedType`（写入非文本类型）。

**待补**：导出的 `New() Clipboard` 构造函数（当前仅有未导出的 `newPlatformClipboard`，外部包无法实例化）。

### 5.2 history 包（持久化层）

**接口**（`store.go`，权威定义）：

```go
type Store interface {
    Append(ctx, e Entry) error                       // 插入/刷新（去重 + 淘汰）
    List(ctx, limit int) ([]Record, error)           // 最近 N 条，新→旧
    Search(ctx, query string, limit int) ([]Record, error) // LIKE 子串搜索
    Clear(ctx) error                                 // 清空全部
    Close() error                                    // 释放数据库句柄
}
```

**实现要点**：
- 纯 Go 驱动 `modernc.org/sqlite`，**不依赖 CGO 工具链**（与 macOS clipboard 的 CGO 解耦，方便交叉编译 Windows/Linux）。
- `Search` 对用户输入做 LIKE 通配符转义（`escapeLike`），防止 `100%` 这类查询匹配全部。
- `parseSQLiteTime` 兼容多种 SQLite 时间格式，解析失败回退 `time.Now()` 而非整查询失败。
- 所有写操作在事务内完成（Append 的 upsert + trim 同事务）。

### 5.3 tray 包（托盘入口）

```go
type Options struct {
    IconPath string                  // PNG/ICO，空则用生成图标
    OnShow, OnClear, OnExit func()   // 菜单回调
}
func New(opts Options) *Tray
func (t *Tray) Run() error           // 阻塞直到退出；macOS 必须在主线程调用
func (t *Tray) Quit()
```

**约束**：systray 在 macOS 上要求运行于主线程，故 `Run()` 由 `main.go` 在主 goroutine 调用，
其余逻辑放后台 goroutine。**当前为 stub，待引入 `getlantern/systray` 实现。**

### 5.4 ui 包（主窗口）

```go
type Deps struct {
    OnSelect func(content string)        // 选中条目 → 回填剪贴板
    OnSearch func(query string) []Item   // 搜索 → 返回展示项
}
type Item struct { Preview, Timestamp string }
func NewWindow(deps Deps) *Window
func (w *Window) Show()  // 显示
func (w *Window) Hide()  // 隐藏到托盘（不销毁）
```

**约束**：窗口关闭语义为「隐藏到托盘」而非退出进程。**当前为 stub，待引入 Fyne 实现。**

### 5.5 main 包（组装与生命周期）

职责（待实现）：
1. 打开 history Store（`history.SQLiteOpen`）。
2. 构造 clipboard 适配器，启动 `Watch` goroutine，消费 out 通道并 `Append` 入库。
3. 构造 ui.Window，接线 `OnSearch → history.Search`、`OnSelect → clipboard.Write`。
4. 构造 tray，接线 `OnShow → window.Show`、`OnClear → history.Clear`、`OnExit → 优雅关闭`。
5. 在主线程 `tray.Run()` 阻塞；退出时 cancel context、Close store。

---

## 6. 技术选型

| 关注点 | 选型 | 理由 |
| --- | --- | --- |
| 语言 | Go 1.26 | 单二进制分发、跨平台交叉编译、并发模型契合后台监听。 |
| 剪贴板（Win/Linux） | `github.com/atotto/clipboard` | 轻量，自动适配 X11/Wayland，无需手写系统调用。 |
| 剪贴板（macOS） | CGO + NSPasteboard | `changeCount` 是 Apple 官方推荐的变更检测方式，轮询开销极低。 |
| 持久化 | `modernc.org/sqlite`（纯 Go） | 无 CGO 依赖，便于交叉编译；SQLite 适合本地结构化历史。 |
| 系统托盘 | `getlantern/systray`（计划） | 跨平台托盘事实标准。 |
| GUI | Fyne（计划） | 纯 Go 跨平台 GUI，与技术栈一致。 |

---

## 7. 跨平台改动范围

| 能力 | macOS | Windows | Linux |
| --- | --- | --- | --- |
| 剪贴板读写 | CGO/AppKit | atotto | atotto |
| 变更监听 | changeCount 轮询 | 哈希轮询 | 哈希轮询 |
| 编译依赖 | 需 Xcode CLT（CGO） | 纯 Go | 纯 Go |
| 托盘图标格式 | PNG | ICO | PNG |
| 配置目录 | `~/Library/Application Support/cut-tool` | `%AppData%/cut-tool` | `~/.config/cut-tool` |

> 注：macOS 因 CGO 无法在非 mac 平台交叉编译；Windows/Linux 目标可在任意平台交叉编译。

---

## 8. 当前已知问题与待办

### 8.1 编译阻塞（需优先处理）

1. **history 包重复声明**：`history.go`（旧 stub）与 `store.go`（新实现）重复定义
   `Store` / `Record`，且签名冲突。`sqlite.go` 依赖 `store.go` 版本，`history.go` 应删除。
2. **缺依赖**：`modernc.org/sqlite` 未加入 `go.mod`，`go build ./...` 报缺包。
3. **clipboard 缺导出构造函数**：仅有未导出的 `newPlatformClipboard`，需补 `New() Clipboard`。

### 8.2 功能待实现

- tray 包接入 `getlantern/systray`。
- ui 包接入 Fyne。
- `main.go` 完整组装与生命周期管理。
- clipboard / history 单元测试。

### 8.3 路线图

| 阶段 | 范围 |
| --- | --- |
| Phase 1（当前） | 包骨架、clipboard 三端实现、history SQLite 实现。 |
| Phase 2 | 修复编译阻塞、补构造函数与单元测试、main 接线打通捕获→入库链路。 |
| Phase 3 | tray + ui GUI 集成，完整交互闭环。 |
| Phase 4（远期） | 图片/文件/富文本类型、全局快捷键、置顶/收藏。 |

---

## 9. 安全与隐私

- 历史数据**仅本地存储**，不做任何网络传输。
- 数据库文件位于用户私有配置目录，权限 `0o755`（目录）。
- 当前**不加密**剪贴板内容；密码类敏感内容会以明文落库，属已知风险，
  后续可考虑敏感内容过滤或加密存储（见 Phase 4 远期项）。

