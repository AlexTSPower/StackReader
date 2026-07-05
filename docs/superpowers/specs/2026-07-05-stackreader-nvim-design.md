# StackReader + StackReader.nvim Implementation Design

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename the `mdv` binary to `StackReader`, add single-file mode and `--watch` auto-reload, then ship a standalone `StackReader.nvim` lazy.nvim-compatible Neovim plugin that opens StackReader as a preview pane alongside editing.

**Architecture:** Two repos — `AlexTSPower/StackReader` (Go binary, existing repo renamed) and `AlexTSPower/StackReader.nvim` (new Lua plugin repo). The binary gains single-file mode and fsnotify-based file watching; the plugin manages nvim terminal buffers running the binary and handles window layout. Neither component knows about the other's internals — the binary reacts to file changes, the plugin spawns processes and manages windows.

**Tech Stack:** Go + bubbletea/glamour/lipgloss/fsnotify (binary); Lua 5.1 + Neovim API (plugin); GitHub Releases API + shell script (binary auto-install).

---

## Global Constraints

- Go module path changes from `github.com/AlexTSPower/mdv` to `github.com/AlexTSPower/StackReader`
- Binary name changes from `mdv` to `stackreader` (lowercase, consistent with Go binary convention)
- Neovim plugin repo: `AlexTSPower/StackReader.nvim`
- Plugin Lua namespace: `stackreader` (e.g. `require("stackreader")`)
- Binary install path: `~/.local/share/nvim/stackreader/bin/stackreader`
- Plugin checks install path first, then falls back to system PATH
- All plugin keymaps are configurable; defaults are `<leader>sp`, `<leader>ss`, `<leader>sb`
- fsnotify version: use the latest stable (`golang.org/x/sys` compatible)
- lazy.nvim `build` hook: `:StackReaderInstall` user command
- **GitHub account:** All repo operations (create, push, release) require the `AlexTSPower` GitHub account. If the active `gh` CLI session is not `AlexTSPower`, run `gh auth switch` before any `gh` command.

---

## Part 1: StackReader Binary Changes

### 1.1 Rename mdv → StackReader

**Scope:** existing repo at `github.com/AlexTSPower/mdv`

- Rename GitHub repo to `StackReader` via GitHub settings (or `gh repo rename`)
- Update Go module path in `go.mod`: `github.com/AlexTSPower/StackReader`
- Update all import paths across `main.go` and `app/` package
- Update `.goreleaser.yaml`: binary name `stackreader`, archive name prefix `stackreader`
- Update `README.md` references

### 1.2 Single-File Mode

**Files:** `main.go`, `app/app.go`, `app/app_test.go`

`main.go` currently rejects non-directory arguments. Change to:
- If the argument is a `.md` or `.mdx` file → call `app.NewSingleFile(path)` instead of `app.New(root)`
- If the argument is a directory → existing `app.New(root)` behaviour unchanged
- If the argument is any other file type → error: `stackreader: not a markdown file or directory`

`app.NewSingleFile(path string) (App, error)`:
- Constructs an App with `showSidebar: false`
- Immediately sends a `FileSelectedMsg{Path: path}` via `Init()` returning the command
- No browser needed; viewer fills full terminal width

### 1.3 `--watch` Flag

**Files:** `main.go`, `app/app.go`, `app/viewer.go`, `app/viewer_test.go`

New flag: `stackreader --watch file.md` (only valid in single-file mode; ignored silently in directory mode).

Implementation:
- Add `fsnotify` dependency: `github.com/fsnotify/fsnotify`
- In `main.go`, parse `--watch` flag before the path argument
- Pass a `watchEnabled bool` into `app.NewSingleFile(path, watch)`
- In `app.go`, if watch is enabled, `Init()` also returns a `watchCmd` that starts a goroutine wrapping `fsnotify.Watcher`
- On `fsnotify.Write` or `fsnotify.Create` event for the watched file, send a `fileChangedMsg{}` to the bubbletea program via `p.Send()`
- In `viewer.go`, handle `fileChangedMsg`: re-read the file from disk and re-render

`fileChangedMsg` is an internal message type in the `app` package.

---

## Part 2: StackReader.nvim Plugin

### 2.1 Repo Structure

New repo: `AlexTSPower/StackReader.nvim`

```
StackReader.nvim/
├── lua/
│   └── stackreader/
│       ├── init.lua         # setup(), resolve_binary(), public API
│       ├── install.lua      # :StackReaderInstall logic, version check
│       ├── preview.lua      # <leader>sp: toggle preview split
│       ├── sidebyside.lua   # <leader>ss: side-by-side edit + preview
│       └── browser.lua      # <leader>sb: directory browser split
├── plugin/
│   └── stackreader.lua      # auto-load: registers user commands + default keymaps
├── scripts/
│   └── install.sh           # platform detection + GitHub release download
└── README.md
```

### 2.2 Binary Resolution (`init.lua`)

`resolve_binary()` returns the path to `stackreader`:
1. Check `~/.local/share/nvim/stackreader/bin/stackreader` — if exists and executable, use it
2. Fall back to `vim.fn.exepath("stackreader")` (system PATH)
3. If neither found, return `nil` and show an error: `"StackReader not installed. Run :StackReaderInstall"`

### 2.3 Binary Install (`install.lua` + `scripts/install.sh`)

`:StackReaderInstall` user command:
- Calls `scripts/install.sh` via `vim.fn.jobstart()`
- Script logic:
  1. Detect OS: `uname -s` → `darwin` or `linux`
  2. Detect arch: `uname -m` → `x86_64` (→ `amd64`) or `arm64`
  3. Fetch latest release tag from `https://api.github.com/repos/AlexTSPower/StackReader/releases/latest`
  4. Construct tarball URL: `https://github.com/AlexTSPower/StackReader/releases/download/<tag>/stackreader_<os>_<arch>.tar.gz`
  5. Download with `curl -sL` to a temp file
  6. Extract `stackreader` binary to `~/.local/share/nvim/stackreader/bin/`
  7. `chmod +x` the binary
- On completion, print: `"StackReader <version> installed successfully"`
- On failure, print the error and exit non-zero

### 2.4 Preview Toggle (`preview.lua`)

`<leader>sp` — toggles a vertical terminal split showing `stackreader --watch <current_file>`.

Behaviour:
- If current buffer is not a `.md` or `.mdx` file, show error: `"Not a markdown file"`
- If preview split already open for this file, close it (toggle off)
- Otherwise: open a vertical split on the right, start terminal with `stackreader --watch <filepath>`, enter terminal mode automatically
- Track open preview windows per-file in a module-level table `{ [filepath] = win_id }`
- When the terminal process exits (user presses `q` in StackReader), the split closes automatically via `TermClose` autocommand

### 2.5 Side-by-Side (`sidebyside.lua`)

`<leader>ss` — opens the current file as a normal nvim buffer on the left, `stackreader --watch <current_file>` on the right.

Behaviour:
- If current buffer is not a markdown file, show error
- If side-by-side already active for this file, close the preview split (toggle off)
- Split: current edit buffer stays left, new vertical split right runs `stackreader --watch <filepath>`
- No BufWritePost autocommand needed — fsnotify in the binary handles sync on save
- On `TermClose` of the stackreader split, clean up the tracking table entry

### 2.6 Browser (`browser.lua`)

`<leader>sb` — opens `stackreader <current_dir>` (no `--watch`) in a vertical split.

`<current_dir>` is `vim.fn.expand("%:p:h")` — the directory of the current buffer's file.

Toggle behaviour: if browser split already open, close it.

### 2.7 `setup()` and Keymaps (`init.lua`, `plugin/stackreader.lua`)

`require("stackreader").setup(opts)` accepts:
```lua
{
  keymaps = {
    preview    = "<leader>sp",   -- toggle preview split
    sidebyside = "<leader>ss",   -- side-by-side edit + preview
    browser    = "<leader>sb",   -- directory browser split
  },
  -- set any keymap to false to disable it
}
```

`plugin/stackreader.lua` auto-loads on nvim start and registers:
- User commands: `:StackReaderInstall`, `:StackReaderPreview`, `:StackReaderSideBySide`, `:StackReaderBrowser`
- Default keymaps (applied unless user calls `setup()` with overrides or `false`)

### 2.8 Health Check

`:checkhealth stackreader` reports:
- Binary path (installed or PATH)
- Binary version (`stackreader --version`)
- Whether fsnotify watch will work (binary version >= the version that added `--watch`)

---

## Workflow Examples

**Read-only preview:**
1. Open any `.md` file in nvim
2. Press `<leader>sp` → vertical split opens with rendered preview
3. Press `<leader>sp` again (or `q` inside StackReader) → split closes

**Edit + live preview:**
1. Open a `.md` file in nvim
2. Press `<leader>ss` → preview split opens on the right
3. Edit in the left buffer, save with `:w` → preview auto-refreshes via fsnotify
4. Press `<leader>ss` again → preview split closes

**Directory browse:**
1. Press `<leader>sb` → StackReader browser opens on the right showing current directory
2. Navigate and select files within StackReader as normal
3. Press `<leader>sb` again or `q` → closes
