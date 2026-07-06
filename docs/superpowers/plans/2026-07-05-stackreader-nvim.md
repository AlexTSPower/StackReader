# StackReader.nvim Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone lazy.nvim-compatible Neovim plugin that auto-installs the StackReader binary and exposes three window modes: preview toggle, side-by-side edit+preview, and directory browser.

**Architecture:** Seven Lua modules behind a single `require("stackreader")` namespace. `plugin/stackreader.lua` auto-loads on nvim start and registers user commands. `init.lua` holds `setup()` and `resolve_binary()`. `install.lua` runs `scripts/install.sh` via `vim.fn.jobstart`. Three mode modules (`preview.lua`, `sidebyside.lua`, `browser.lua`) each manage their own window-tracking table and expose a single `toggle()` function. `health.lua` powers `:checkhealth stackreader`.

**Tech Stack:** Lua 5.1, Neovim 0.10+ API (`vim.api`, `vim.fn`, `vim.system`, `vim.health`), bash install script, GitHub Releases API.

## Global Constraints

- **Prerequisite:** The StackReader binary plan (`2026-07-05-stackreader-binary.md`) must be complete and a GitHub release published before this plan can be fully tested. The install script downloads from GitHub releases.
- Repo name: `AlexTSPower/StackReader.nvim`
- Lua namespace: `stackreader` — all modules live under `lua/stackreader/`
- Binary install path: `~/.local/share/nvim/stackreader/bin/stackreader`
- `resolve_binary()` checks install path first, then `vim.fn.exepath("stackreader")`
- Default keymaps: `<leader>sp` (preview), `<leader>ss` (side-by-side), `<leader>sb` (browser)
- Setting a keymap to `false` in `setup()` disables it entirely
- Targets Neovim 0.10+ (uses `vim.system`, `vim.health.start`)
- **GitHub account:** All `gh` commands require the `AlexTSPower` account. Run `gh auth switch` first if needed. Verify with `gh auth status`.

---

### Task 1: Repo scaffold + install.sh

**Files:**
- Create: `scripts/install.sh`
- Create: `README.md` (skeleton — completed in Task 7)
- Create: `.gitignore`

**Interfaces:**
- Produces: `scripts/install.sh` — callable as `bash scripts/install.sh`; installs binary to `~/.local/share/nvim/stackreader/bin/stackreader`

- [ ] **Step 1: Verify gh auth and create repo**

```bash
gh auth status
```
Expected: shows `AlexTSPower`. If not, run `gh auth switch`.

```bash
gh repo create AlexTSPower/StackReader.nvim --public --description "Neovim plugin for StackReader markdown viewer"
```

- [ ] **Step 2: Clone the new repo**

```bash
cd ~/Accounts/PersonalProjects
git clone git@github.com-accenture:AlexTSPower/StackReader.nvim.git
cd StackReader.nvim
```

- [ ] **Step 3: Create directory structure**

```bash
mkdir -p lua/stackreader scripts plugin
```

- [ ] **Step 4: Write scripts/install.sh**

Create `scripts/install.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${HOME}/.local/share/nvim/stackreader/bin"
mkdir -p "${INSTALL_DIR}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "${OS}" in
  darwin) OS="darwin" ;;
  linux)  OS="linux"  ;;
  *) echo "Unsupported OS: ${OS}" >&2; exit 1 ;;
esac

# Detect arch
ARCH=$(uname -m)
case "${ARCH}" in
  x86_64)        ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported arch: ${ARCH}" >&2; exit 1 ;;
esac

# Fetch latest release tag from GitHub API
API_URL="https://api.github.com/repos/AlexTSPower/StackReader/releases/latest"
TAG=$(curl -fsSL "${API_URL}" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "${TAG}" ]; then
  echo "Failed to fetch latest release tag from GitHub" >&2
  exit 1
fi

VERSION="${TAG#v}"
TARBALL="stackreader_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/AlexTSPower/StackReader/releases/download/${TAG}/${TARBALL}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "${TMPDIR}"' EXIT

echo "Downloading ${TARBALL}..."
curl -fsSL -o "${TMPDIR}/${TARBALL}" "${URL}"

echo "Extracting..."
tar -xzf "${TMPDIR}/${TARBALL}" -C "${TMPDIR}"

cp "${TMPDIR}/stackreader" "${INSTALL_DIR}/stackreader"
chmod +x "${INSTALL_DIR}/stackreader"

echo "StackReader ${TAG} installed to ${INSTALL_DIR}/stackreader"
```

Make it executable:
```bash
chmod +x scripts/install.sh
```

- [ ] **Step 5: Write skeleton README.md**

Create `README.md`:
```markdown
# StackReader.nvim

Neovim plugin for [StackReader](https://github.com/AlexTSPower/StackReader) — a terminal markdown viewer with GitHub-style rendering.

> Full documentation coming in v0.1.0.
```

- [ ] **Step 6: Write .gitignore**

Create `.gitignore`:
```
.DS_Store
```

- [ ] **Step 7: Test install.sh manually**

```bash
bash scripts/install.sh
```

Expected output (approximately):
```
Downloading stackreader_0.3.2_darwin_arm64.tar.gz...
Extracting...
StackReader v0.3.2 installed to /Users/<you>/.local/share/nvim/stackreader/bin/stackreader
```

Verify:
```bash
~/.local/share/nvim/stackreader/bin/stackreader --version
```
Expected: `stackreader v0.3.2` (or current release version)

- [ ] **Step 8: Initial commit and push**

```bash
git add scripts/install.sh README.md .gitignore
git commit -m "chore: repo scaffold and binary install script"
git push -u origin main
```

---

### Task 2: init.lua + install.lua

**Files:**
- Create: `lua/stackreader/init.lua` — `setup()`, `resolve_binary()`, `config` table
- Create: `lua/stackreader/install.lua` — `:StackReaderInstall` logic

**Interfaces:**
- Produces:
  - `require("stackreader").resolve_binary()` → `string|nil` — path to binary or nil
  - `require("stackreader").setup(opts)` — merges config, registers keymaps
  - `require("stackreader").config` — table with `keymaps.preview`, `keymaps.sidebyside`, `keymaps.browser`
  - `require("stackreader.install").install()` — runs install.sh async

- [ ] **Step 1: Write lua/stackreader/init.lua**

```lua
local M = {}

M.config = {
  keymaps = {
    preview    = "<leader>sp",
    sidebyside = "<leader>ss",
    browser    = "<leader>sb",
  },
}

-- Returns path to the stackreader binary, or nil if not found.
function M.resolve_binary()
  local installed = vim.fn.expand("~/.local/share/nvim/stackreader/bin/stackreader")
  if vim.fn.executable(installed) == 1 then
    return installed
  end
  local system_path = vim.fn.exepath("stackreader")
  if system_path ~= "" then
    return system_path
  end
  return nil
end

-- setup() merges user opts into config and registers keymaps.
-- Call from your lazy.nvim config function.
function M.setup(opts)
  opts = opts or {}
  if opts.keymaps then
    for k, v in pairs(opts.keymaps) do
      M.config.keymaps[k] = v
    end
  end

  local km = M.config.keymaps

  if km.preview ~= false then
    vim.keymap.set("n", km.preview, function()
      require("stackreader.preview").toggle()
    end, { desc = "StackReader: toggle preview" })
  end

  if km.sidebyside ~= false then
    vim.keymap.set("n", km.sidebyside, function()
      require("stackreader.sidebyside").toggle()
    end, { desc = "StackReader: side-by-side" })
  end

  if km.browser ~= false then
    vim.keymap.set("n", km.browser, function()
      require("stackreader.browser").toggle()
    end, { desc = "StackReader: browser" })
  end
end

return M
```

- [ ] **Step 2: Write lua/stackreader/install.lua**

```lua
local M = {}

function M.install()
  -- Find install.sh relative to this file's location in the plugin.
  local sources = vim.api.nvim_get_runtime_file("lua/stackreader/install.lua", false)
  if #sources == 0 then
    vim.notify("StackReader: cannot locate plugin directory", vim.log.levels.ERROR)
    return
  end
  -- sources[1] is .../StackReader.nvim/lua/stackreader/install.lua
  -- go up 3 dirs: stackreader/ -> lua/ -> StackReader.nvim/
  local plugin_dir = vim.fn.fnamemodify(sources[1], ":h:h:h")
  local script = plugin_dir .. "/scripts/install.sh"

  if vim.fn.filereadable(script) == 0 then
    vim.notify("StackReader: install.sh not found at " .. script, vim.log.levels.ERROR)
    return
  end

  vim.notify("StackReader: installing binary...", vim.log.levels.INFO)

  vim.fn.jobstart({ "bash", script }, {
    on_stdout = function(_, data)
      for _, line in ipairs(data) do
        if line ~= "" then
          vim.notify(line, vim.log.levels.INFO)
        end
      end
    end,
    on_stderr = function(_, data)
      for _, line in ipairs(data) do
        if line ~= "" then
          vim.notify(line, vim.log.levels.WARN)
        end
      end
    end,
    on_exit = function(_, code)
      if code ~= 0 then
        vim.notify(
          "StackReader: install failed (exit " .. code .. ")",
          vim.log.levels.ERROR
        )
      end
    end,
  })
end

return M
```

- [ ] **Step 3: Verify in Neovim (manual)**

Add the plugin to your lazy.nvim config temporarily:
```lua
{ dir = "~/Accounts/PersonalProjects/StackReader.nvim" }
```

Restart nvim, then:
```
:lua print(require("stackreader").resolve_binary())
```
Expected: prints the path to the binary (e.g. `/Users/you/.local/share/nvim/stackreader/bin/stackreader`)

```
:lua require("stackreader.install").install()
```
Expected: notifications show download progress; binary re-installed.

- [ ] **Step 4: Commit**

```bash
git add lua/stackreader/init.lua lua/stackreader/install.lua
git commit -m "feat: add init.lua (resolve_binary, setup) and install.lua"
```

---

### Task 3: plugin/stackreader.lua — user commands

**Files:**
- Create: `plugin/stackreader.lua` — user commands auto-loaded by Neovim on startup

**Interfaces:**
- Consumes:
  - `require("stackreader.install").install()`
  - `require("stackreader.preview").toggle()`
  - `require("stackreader.sidebyside").toggle()`
  - `require("stackreader.browser").toggle()`
- Produces: user commands `:StackReaderInstall`, `:StackReaderPreview`, `:StackReaderSideBySide`, `:StackReaderBrowser`

- [ ] **Step 1: Write plugin/stackreader.lua**

```lua
-- Auto-loaded by Neovim on startup. Registers user commands.
-- Keymaps are registered separately in setup() so users can customise them.

vim.api.nvim_create_user_command("StackReaderInstall", function()
  require("stackreader.install").install()
end, { desc = "Install or update the StackReader binary" })

vim.api.nvim_create_user_command("StackReaderPreview", function()
  require("stackreader.preview").toggle()
end, { desc = "Toggle StackReader preview split for current file" })

vim.api.nvim_create_user_command("StackReaderSideBySide", function()
  require("stackreader.sidebyside").toggle()
end, { desc = "Toggle StackReader side-by-side edit + preview" })

vim.api.nvim_create_user_command("StackReaderBrowser", function()
  require("stackreader.browser").toggle()
end, { desc = "Open StackReader directory browser" })
```

- [ ] **Step 2: Verify commands exist in Neovim (manual)**

Restart nvim, then:
```
:StackReaderInstall
```
Expected: runs install (notification appears).

```
:command StackReader
```
Expected: lists all four StackReader commands.

- [ ] **Step 3: Commit**

```bash
git add plugin/stackreader.lua
git commit -m "feat: register user commands in plugin/stackreader.lua"
```

---

### Task 4: preview.lua — toggle preview split

**Files:**
- Create: `lua/stackreader/preview.lua`

**Interfaces:**
- Consumes: `require("stackreader").resolve_binary()` → binary path
- Produces: `require("stackreader.preview").toggle()` — opens/closes preview split

- [ ] **Step 1: Write lua/stackreader/preview.lua**

```lua
local M = {}

-- Tracks open preview windows: { [filepath] = win_id }
local windows = {}

local function is_markdown(filepath)
  local ext = vim.fn.fnamemodify(filepath, ":e"):lower()
  return ext == "md" or ext == "mdx"
end

function M.toggle()
  local binary = require("stackreader").resolve_binary()
  if not binary then
    vim.notify(
      "StackReader not installed. Run :StackReaderInstall",
      vim.log.levels.ERROR
    )
    return
  end

  local filepath = vim.fn.expand("%:p")
  if filepath == "" or not is_markdown(filepath) then
    vim.notify("StackReader: not a markdown file", vim.log.levels.WARN)
    return
  end

  -- Toggle off: close the existing window.
  local existing_win = windows[filepath]
  if existing_win and vim.api.nvim_win_is_valid(existing_win) then
    vim.api.nvim_win_close(existing_win, true)
    windows[filepath] = nil
    return
  end

  -- Open a vertical split to the right.
  vim.cmd("vsplit")
  local win = vim.api.nvim_get_current_win()
  windows[filepath] = win

  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_win_set_buf(win, buf)

  vim.fn.termopen({ binary, "--watch", filepath }, {
    on_exit = function()
      windows[filepath] = nil
      if vim.api.nvim_win_is_valid(win) then
        vim.api.nvim_win_close(win, true)
      end
    end,
  })

  vim.cmd("startinsert")
end

return M
```

- [ ] **Step 2: Verify in Neovim (manual)**

Open any `.md` file:
```
:e ~/some/file.md
```

Press `<leader>sp` (or run `:StackReaderPreview`).
Expected: vertical split opens on the right showing the rendered markdown. StackReader is in `--watch` mode.

Edit and save the file in another window (`:w`).
Expected: the preview refreshes.

Press `<leader>sp` again.
Expected: preview split closes.

Press `q` inside the StackReader split.
Expected: split closes automatically.

- [ ] **Step 3: Commit**

```bash
git add lua/stackreader/preview.lua
git commit -m "feat: add preview.lua — toggle rendered preview split"
```

---

### Task 5: sidebyside.lua — edit + preview layout

**Files:**
- Create: `lua/stackreader/sidebyside.lua`

**Interfaces:**
- Consumes: `require("stackreader").resolve_binary()` → binary path
- Produces: `require("stackreader.sidebyside").toggle()` — opens/closes side-by-side layout; edit window retains focus

- [ ] **Step 1: Write lua/stackreader/sidebyside.lua**

```lua
local M = {}

-- Tracks open preview windows for side-by-side mode: { [filepath] = win_id }
local windows = {}

local function is_markdown(filepath)
  local ext = vim.fn.fnamemodify(filepath, ":e"):lower()
  return ext == "md" or ext == "mdx"
end

function M.toggle()
  local binary = require("stackreader").resolve_binary()
  if not binary then
    vim.notify(
      "StackReader not installed. Run :StackReaderInstall",
      vim.log.levels.ERROR
    )
    return
  end

  local filepath = vim.fn.expand("%:p")
  if filepath == "" or not is_markdown(filepath) then
    vim.notify("StackReader: not a markdown file", vim.log.levels.WARN)
    return
  end

  -- Toggle off: close the preview split, keep the edit buffer.
  local existing_win = windows[filepath]
  if existing_win and vim.api.nvim_win_is_valid(existing_win) then
    vim.api.nvim_win_close(existing_win, true)
    windows[filepath] = nil
    return
  end

  -- Remember the edit window so we can return focus to it.
  local edit_win = vim.api.nvim_get_current_win()

  -- Open preview split to the right.
  vim.cmd("vsplit")
  local preview_win = vim.api.nvim_get_current_win()
  windows[filepath] = preview_win

  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_win_set_buf(preview_win, buf)

  vim.fn.termopen({ binary, "--watch", filepath }, {
    on_exit = function()
      windows[filepath] = nil
      if vim.api.nvim_win_is_valid(preview_win) then
        vim.api.nvim_win_close(preview_win, true)
      end
    end,
  })

  -- Return focus to the edit buffer so the user can keep typing.
  vim.api.nvim_set_current_win(edit_win)
end

return M
```

- [ ] **Step 2: Verify in Neovim (manual)**

Open any `.md` file and press `<leader>ss` (or `:StackReaderSideBySide`).
Expected:
- Left window: your editable markdown buffer (cursor stays here)
- Right window: StackReader preview in `--watch` mode

Edit the file, save with `:w`.
Expected: preview on the right refreshes automatically.

Press `<leader>ss` again.
Expected: right split closes, left edit buffer remains.

- [ ] **Step 3: Commit**

```bash
git add lua/stackreader/sidebyside.lua
git commit -m "feat: add sidebyside.lua — edit + preview with focus on edit buffer"
```

---

### Task 6: browser.lua — directory browser split

**Files:**
- Create: `lua/stackreader/browser.lua`

**Interfaces:**
- Consumes: `require("stackreader").resolve_binary()` → binary path
- Produces: `require("stackreader.browser").toggle()` — opens/closes browser split running `stackreader <dir>`

- [ ] **Step 1: Write lua/stackreader/browser.lua**

```lua
local M = {}

-- Tracks open browser windows: { [dirpath] = win_id }
local windows = {}

function M.toggle()
  local binary = require("stackreader").resolve_binary()
  if not binary then
    vim.notify(
      "StackReader not installed. Run :StackReaderInstall",
      vim.log.levels.ERROR
    )
    return
  end

  -- Use the directory of the current buffer's file.
  local dirpath = vim.fn.expand("%:p:h")
  if dirpath == "" then
    dirpath = vim.fn.getcwd()
  end

  -- Toggle off: close the existing browser window.
  local existing_win = windows[dirpath]
  if existing_win and vim.api.nvim_win_is_valid(existing_win) then
    vim.api.nvim_win_close(existing_win, true)
    windows[dirpath] = nil
    return
  end

  -- Open vertical split on the right.
  local edit_win = vim.api.nvim_get_current_win()
  vim.cmd("vsplit")
  local win = vim.api.nvim_get_current_win()
  windows[dirpath] = win

  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_win_set_buf(win, buf)

  -- No --watch: browser mode, user navigates within StackReader.
  vim.fn.termopen({ binary, dirpath }, {
    on_exit = function()
      windows[dirpath] = nil
      if vim.api.nvim_win_is_valid(win) then
        vim.api.nvim_win_close(win, true)
      end
    end,
  })

  vim.cmd("startinsert")
end

return M
```

- [ ] **Step 2: Verify in Neovim (manual)**

Press `<leader>sb` (or `:StackReaderBrowser`).
Expected: vertical split opens showing StackReader's file browser for the current directory.

Navigate files within StackReader, select a file.
Expected: file opens in StackReader's viewer within the split.

Press `q` inside StackReader.
Expected: split closes automatically.

Press `<leader>sb` again while split is open.
Expected: split closes.

- [ ] **Step 3: Commit**

```bash
git add lua/stackreader/browser.lua
git commit -m "feat: add browser.lua — directory browser split"
```

---

### Task 7: Health check + complete README + v0.1.0 release

**Files:**
- Create: `lua/stackreader/health.lua`
- Modify: `README.md` — complete installation and usage docs

**Interfaces:**
- Produces: `:checkhealth stackreader` reporting binary path and version

- [ ] **Step 1: Write lua/stackreader/health.lua**

```lua
local M = {}

function M.check()
  vim.health.start("StackReader")

  local binary = require("stackreader").resolve_binary()

  if not binary then
    vim.health.error(
      "stackreader binary not found",
      {
        "Run :StackReaderInstall to download the binary automatically",
        "Or install manually: brew install AlexTSPower/tap/stackreader",
      }
    )
    return
  end

  vim.health.ok("binary found: " .. binary)

  local result = vim.system({ binary, "--version" }, { text = true }):wait()
  if result.code == 0 then
    vim.health.ok("version: " .. vim.trim(result.stdout))
  else
    vim.health.warn(
      "could not determine version",
      result.stderr ~= "" and result.stderr or "unknown error"
    )
  end
end

return M
```

- [ ] **Step 2: Verify :checkhealth stackreader in Neovim (manual)**

```
:checkhealth stackreader
```

Expected output:
```
StackReader ~
  OK binary found: /Users/<you>/.local/share/nvim/stackreader/bin/stackreader
  OK version: stackreader v0.3.2
```

- [ ] **Step 3: Write complete README.md**

Replace `README.md` with:
```markdown
# StackReader.nvim

A Neovim plugin for [StackReader](https://github.com/AlexTSPower/StackReader) — a terminal markdown viewer with GitHub-style rendering.

## Requirements

- Neovim 0.10+
- `curl` and `tar` (used by the auto-installer)

## Installation

### lazy.nvim

```lua
{
  "AlexTSPower/StackReader.nvim",
  build = ":StackReaderInstall",
  config = function()
    require("stackreader").setup({
      keymaps = {
        preview    = "<leader>sp",
        sidebyside = "<leader>ss",
        browser    = "<leader>sb",
      },
    })
  end,
}
```

Run `:checkhealth stackreader` after install to confirm the binary is ready.

## Usage

| Keymap | Command | Description |
|--------|---------|-------------|
| `<leader>sp` | `:StackReaderPreview` | Toggle rendered preview alongside current buffer |
| `<leader>ss` | `:StackReaderSideBySide` | Side-by-side: edit on left, live preview on right |
| `<leader>sb` | `:StackReaderBrowser` | Open markdown browser for current directory |

Set any keymap to `false` to disable it:

```lua
require("stackreader").setup({
  keymaps = { browser = false }
})
```

## Manual Binary Install

If `:StackReaderInstall` fails:

```sh
brew install AlexTSPower/tap/stackreader
```

Or download from [GitHub Releases](https://github.com/AlexTSPower/StackReader/releases) and place the `stackreader` binary anywhere on your `$PATH`.

## How it works

StackReader.nvim opens the [`stackreader`](https://github.com/AlexTSPower/StackReader) binary inside Neovim terminal buffers. In preview and side-by-side modes, `--watch` is passed so the binary uses `fsnotify` to detect file saves and re-render automatically — no polling, no BufWritePost hooks needed.

## License

MIT
```

- [ ] **Step 4: Commit README and health check**

```bash
git add lua/stackreader/health.lua README.md
git commit -m "feat: add health check and complete README"
```

- [ ] **Step 5: Tag v0.1.0 and push**

```bash
git tag v0.1.0
git push origin main --tags
```

Expected: tag published at `https://github.com/AlexTSPower/StackReader.nvim/releases/tag/v0.1.0` (GitHub auto-creates a release for the tag, or create one manually with `gh release create v0.1.0 --title "v0.1.0" --notes "Initial release"`).

- [ ] **Step 6: Verify lazy.nvim install end-to-end (manual)**

Add to a test nvim config:
```lua
{
  "AlexTSPower/StackReader.nvim",
  build = ":StackReaderInstall",
  config = function()
    require("stackreader").setup()
  end,
}
```

Run `:Lazy sync`, then `:checkhealth stackreader`.
Expected: binary found, version reported.

Open a `.md` file and verify all three keymaps work:
- `<leader>sp` → preview toggle
- `<leader>ss` → side-by-side
- `<leader>sb` → browser
