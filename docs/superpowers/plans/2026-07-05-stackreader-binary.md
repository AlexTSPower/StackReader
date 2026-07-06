# StackReader Binary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the `mdv` binary to `stackreader`, add single-file mode so `stackreader file.md` opens directly in the viewer, and add `--watch` so the viewer auto-reloads when the file changes on disk.

**Architecture:** Three sequential changes to the existing Go repo. The rename is mechanical (module path + goreleaser config). Single-file mode adds `NewSingleFile(path, watch bool)` to `app.go` and updates `main.go` to detect file arguments. The watch feature adds `fsnotify`, stores a `*fsnotify.Watcher` in the App struct, and uses bubbletea's `tea.Cmd` pattern to relay file-change events back into the update loop.

**Tech Stack:** Go 1.26, bubbletea, glamour, lipgloss, fsnotify v1.

## Global Constraints

- Go module path: `github.com/AlexTSPower/StackReader` (was `github.com/AlexTSPower/mdv`)
- Binary name in goreleaser: `stackreader` (lowercase)
- Archive name template: `stackreader_{{ .Version }}_{{ .Os }}_{{ .Arch }}`
- All error prefix strings change from `mdv:` to `stackreader:`
- fsnotify dependency: `github.com/fsnotify/fsnotify` latest stable
- `--watch` is silently ignored when opening a directory (no error)
- **GitHub account:** All `gh` commands require the `AlexTSPower` account. Run `gh auth switch` first if the active session is a different account. Verify with `gh auth status`.

---

### Task 1: Rename mdv → StackReader

**Files:**
- Modify: `go.mod` — module path
- Modify: `main.go` — import path + error prefix strings
- Modify: `.goreleaser.yaml` — binary name, archive template, brew formula
- Modify: `README.md` — any `mdv` references → `stackreader`

**Interfaces:**
- Produces: Go module path `github.com/AlexTSPower/StackReader`, binary name `stackreader`; all later tasks use this module path

- [ ] **Step 1: Verify gh auth is AlexTSPower**

```bash
gh auth status
```
Expected: shows `AlexTSPower` as the active account. If not, run `gh auth switch` and select the AlexTSPower account.

- [ ] **Step 2: Rename GitHub repo**

```bash
gh repo rename StackReader --repo AlexTSPower/mdv
```
Expected: `✓ Renamed repository AlexTSPower/mdv to AlexTSPower/StackReader`

- [ ] **Step 3: Update git remote URL**

```bash
git remote set-url origin git@github.com-accenture:AlexTSPower/StackReader.git
git remote -v
```
Expected: both fetch and push show `github.com-accenture:AlexTSPower/StackReader.git`

- [ ] **Step 4: Update go.mod module path**

```bash
go mod edit -module github.com/AlexTSPower/StackReader
```

Then verify `go.mod` first line reads:
```
module github.com/AlexTSPower/StackReader
```

- [ ] **Step 5: Update import path in main.go**

In `main.go`, change:
```go
"github.com/AlexTSPower/mdv/app"
```
to:
```go
"github.com/AlexTSPower/StackReader/app"
```

Also change all error prefix strings from `mdv:` to `stackreader:`:
```go
fmt.Fprintf(os.Stderr, "stackreader: %v\n", err)
// and
fmt.Fprintf(os.Stderr, "stackreader: %s is not a directory\n", root)
```

Full updated `main.go`:
```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/AlexTSPower/StackReader/app"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("stackreader", version)
		return
	}

	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stackreader: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "stackreader: %s is not a directory\n", root)
		os.Exit(1)
	}

	model, err := app.New(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stackreader: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "stackreader: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Update .goreleaser.yaml**

Replace the entire `.goreleaser.yaml` with:
```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    main: .
    binary: stackreader
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - formats:
      - tar.gz
    name_template: "stackreader_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md

checksum:
  name_template: checksums.txt

brews:
  - repository:
      owner: AlexTSPower
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: https://github.com/AlexTSPower/StackReader
    description: Terminal markdown viewer with GitHub-style rendering
    license: MIT
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com
    directory: Formula
    install: |
      bin.install "stackreader"
    test: |
      system "#{bin}/stackreader", "--version"
```

- [ ] **Step 7: Update README.md**

Replace all occurrences of `mdv` with `stackreader` and `AlexTSPower/mdv` with `AlexTSPower/StackReader` in `README.md`. Use:

```bash
sed -i '' 's/AlexTSPower\/mdv/AlexTSPower\/StackReader/g' README.md
sed -i '' 's/`mdv`/`stackreader`/g' README.md
sed -i '' 's/ mdv / stackreader /g' README.md
```

Read the file after to verify it looks correct.

- [ ] **Step 8: Verify build and tests pass**

```bash
go build ./...
go test ./...
```

Expected: all tests pass, no build errors.

- [ ] **Step 9: Commit and push**

```bash
git add go.mod main.go .goreleaser.yaml README.md
git commit -m "chore: rename mdv to stackreader"
git push origin master
```

---

### Task 2: Single-file mode

**Files:**
- Modify: `app/app.go` — add `NewSingleFile`, `singleFile` field, update `Init`, handle `b` key noop in single-file mode
- Modify: `main.go` — detect file vs directory argument, call `NewSingleFile`
- Modify: `app/app_test.go` — tests for NewSingleFile behaviour

**Interfaces:**
- Consumes: `app.New(root string) (App, error)` — unchanged
- Produces: `app.NewSingleFile(path string, watch bool) (App, error)` — Task 3 passes `watch=true`; Task 2 only tests `watch=false` behaviour

- [ ] **Step 1: Write failing tests for NewSingleFile**

Add to `app/app_test.go`:
```go
func TestApp_NewSingleFile_NoSidebar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello"), 0644)

	a, err := NewSingleFile(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if a.showSidebar {
		t.Error("single-file mode should have showSidebar=false")
	}
	if a.focus != focusViewer {
		t.Error("single-file mode should start focused on viewer")
	}
	if a.singleFile != path {
		t.Errorf("singleFile should be %q, got %q", path, a.singleFile)
	}
}

func TestApp_NewSingleFile_InitSendsFileSelectedMsg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello"), 0644)

	a, err := NewSingleFile(path, false)
	if err != nil {
		t.Fatal(err)
	}
	cmd := a.Init()
	if cmd == nil {
		t.Fatal("Init() should return a command in single-file mode")
	}
	msg := cmd()
	sel, ok := msg.(FileSelectedMsg)
	if !ok {
		t.Fatalf("Init() command should return FileSelectedMsg, got %T", msg)
	}
	if sel.Path != path {
		t.Errorf("FileSelectedMsg.Path = %q, want %q", sel.Path, path)
	}
}

func TestApp_SingleFileMode_BKeyIsNoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello"), 0644)

	a, err := NewSingleFile(path, false)
	if err != nil {
		t.Fatal(err)
	}
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a2 := model.(App)

	model, _ = a2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	a3 := model.(App)
	if a3.showSidebar {
		t.Error("b key should not toggle sidebar in single-file mode")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/ -run "TestApp_NewSingleFile|TestApp_SingleFileMode" -v
```

Expected: `FAIL` — `NewSingleFile` undefined, `singleFile` field undefined.

- [ ] **Step 3: Add singleFile field and NewSingleFile to app/app.go**

Add `singleFile string` to the `App` struct (after `statusMsg`):
```go
type App struct {
	browser     Browser
	viewer      Viewer
	width       int
	height      int
	showSidebar bool
	focus       focusTarget
	currentFile string
	statusMsg   string
	singleFile  string // non-empty when started with a file argument
}
```

Add `NewSingleFile` function after `New`:
```go
// NewSingleFile constructs an App in single-file mode: no browser, viewer fills
// the full terminal. watch is stored for use in Task 3; pass false for now.
func NewSingleFile(path string, watch bool) (App, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return App{}, err
	}
	return App{
		viewer:      NewViewer(80, 20),
		showSidebar: false,
		focus:       focusViewer,
		singleFile:  absPath,
	}, nil
}
```

- [ ] **Step 4: Update Init() to send FileSelectedMsg in single-file mode**

Replace the existing `Init()`:
```go
func (a App) Init() tea.Cmd {
	if a.singleFile == "" {
		return nil
	}
	path := a.singleFile
	return func() tea.Msg { return FileSelectedMsg{Path: path} }
}
```

- [ ] **Step 5: Make the b key a noop in single-file mode**

In `Update`, in the `case "b":` branch, add a guard at the top:
```go
case "b":
	if a.singleFile != "" {
		return a, nil
	}
	a.showSidebar = !a.showSidebar
	a = a.applyLayout()
	return a, nil
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./app/ -run "TestApp_NewSingleFile|TestApp_SingleFileMode" -v
```

Expected: all three new tests PASS.

- [ ] **Step 7: Run full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 8: Update main.go to detect file arguments**

Replace the entire `main.go` with:
```go
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/AlexTSPower/StackReader/app"
)

var version = "dev"

func main() {
	// Collect non-flag args; handle --version/-v early.
	var watchFlag bool
	var paths []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v":
			fmt.Println("stackreader", version)
			return
		case "--watch":
			watchFlag = true
		default:
			paths = append(paths, arg)
		}
	}
	_ = watchFlag // used in Task 3

	path := "."
	if len(paths) > 0 {
		path = paths[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stackreader: %v\n", err)
		os.Exit(1)
	}

	var model tea.Model
	if info.IsDir() {
		model, err = app.New(path)
	} else {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".mdx" {
			fmt.Fprintf(os.Stderr, "stackreader: not a markdown file or directory\n")
			os.Exit(1)
		}
		model, err = app.NewSingleFile(path, false) // watch wired in Task 3
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "stackreader: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "stackreader: %v\n", err)
		os.Exit(1)
	}
}
```

Note: this requires adding `"path/filepath"` to the imports. Full import block:
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/AlexTSPower/StackReader/app"
)
```

- [ ] **Step 9: Build and verify**

```bash
go build ./...
go test ./...
```

Expected: compiles, all tests pass.

- [ ] **Step 10: Commit**

```bash
git add app/app.go app/app_test.go main.go
git commit -m "feat: add single-file mode — stackreader file.md opens viewer directly"
```

---

### Task 3: --watch flag + release

**Files:**
- Modify: `go.mod` / `go.sum` — fsnotify dependency
- Modify: `app/app.go` — `watcher *fsnotify.Watcher` field, `fileChangedMsg`, `watchFileCmd`, updated `Init()`, updated `NewSingleFile`, `fileChangedMsg` case in `Update`
- Modify: `app/app_test.go` — `fileChangedMsg` reload test
- Modify: `main.go` — pass `watchFlag` to `NewSingleFile`

**Interfaces:**
- Consumes: `app.NewSingleFile(path string, watch bool)` from Task 2 (signature already accepts `watch bool`, just unused)
- Produces: `stackreader --watch file.md` auto-reloads on save; triggers goreleaser release

- [ ] **Step 1: Add fsnotify dependency**

```bash
go get github.com/fsnotify/fsnotify
go mod tidy
```

Expected: `go.mod` now lists `github.com/fsnotify/fsnotify` as a direct dependency.

- [ ] **Step 2: Write failing test for fileChangedMsg**

Add to `app/app_test.go`:
```go
func TestApp_FileChangedMsg_ReloadsViewer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Version 1"), 0644)

	a, err := NewSingleFile(path, false)
	if err != nil {
		t.Fatal(err)
	}
	// Apply dimensions so the viewer is usable.
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a2 := model.(App)

	// Simulate file load via Init message.
	model, _ = a2.Update(FileSelectedMsg{Path: path})
	a3 := model.(App)

	// Update the file on disk.
	os.WriteFile(path, []byte("# Version 2"), 0644)

	// Send fileChangedMsg — viewer should reload from disk.
	model, _ = a3.Update(fileChangedMsg{})
	a4 := model.(App)

	view := a4.viewer.View()
	if !strings.Contains(view, "Version 2") {
		t.Errorf("viewer should show updated content after fileChangedMsg, got: %q", view)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./app/ -run TestApp_FileChangedMsg_ReloadsViewer -v
```

Expected: `FAIL` — `fileChangedMsg` undefined.

- [ ] **Step 4: Add fileChangedMsg, watcher field, watchFileCmd to app/app.go**

Add to the import block (add `"github.com/fsnotify/fsnotify"`):
```go
import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
```

Add `fileChangedMsg` type (near `clearStatusMsg`):
```go
// fileChangedMsg is sent by the fsnotify watcher when the open file changes.
type fileChangedMsg struct{}
```

Add `watcher *fsnotify.Watcher` to the `App` struct (after `singleFile`):
```go
type App struct {
	browser     Browser
	viewer      Viewer
	width       int
	height      int
	showSidebar bool
	focus       focusTarget
	currentFile string
	statusMsg   string
	singleFile  string
	watcher     *fsnotify.Watcher // non-nil when --watch is active
}
```

Add `watchFileCmd` helper after `setStatus`:
```go
// watchFileCmd returns a Cmd that blocks until a Write or Create event fires on
// the watched file, then returns fileChangedMsg. Must be re-issued after each
// event to keep the watch loop running.
func watchFileCmd(w *fsnotify.Watcher) tea.Cmd {
	return func() tea.Msg {
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return nil
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					return fileChangedMsg{}
				}
			case _, ok := <-w.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}
```

- [ ] **Step 5: Update NewSingleFile to create watcher when watch=true**

Replace `NewSingleFile`:
```go
func NewSingleFile(path string, watch bool) (App, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return App{}, err
	}
	a := App{
		viewer:      NewViewer(80, 20),
		showSidebar: false,
		focus:       focusViewer,
		singleFile:  absPath,
	}
	if watch {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			return App{}, err
		}
		if err := w.Add(absPath); err != nil {
			w.Close()
			return App{}, err
		}
		a.watcher = w
	}
	return a, nil
}
```

- [ ] **Step 6: Update Init() to also start the watch loop**

Replace `Init()`:
```go
func (a App) Init() tea.Cmd {
	if a.singleFile == "" {
		return nil
	}
	path := a.singleFile
	cmds := []tea.Cmd{
		func() tea.Msg { return FileSelectedMsg{Path: path} },
	}
	if a.watcher != nil {
		cmds = append(cmds, watchFileCmd(a.watcher))
	}
	return tea.Batch(cmds...)
}
```

- [ ] **Step 7: Handle fileChangedMsg in Update**

In the `Update` switch, add a case after `clearStatusMsg`:
```go
case fileChangedMsg:
	// Re-read the file and re-render, then re-arm the watcher.
	var cmd tea.Cmd
	a.viewer, cmd = a.viewer.Update(FileSelectedMsg{Path: a.singleFile})
	var wCmd tea.Cmd
	if a.watcher != nil {
		wCmd = watchFileCmd(a.watcher)
	}
	return a, tea.Batch(cmd, wCmd)
```

- [ ] **Step 8: Run test to verify it passes**

```bash
go test ./app/ -run TestApp_FileChangedMsg_ReloadsViewer -v
```

Expected: PASS.

- [ ] **Step 9: Wire --watch flag in main.go**

In `main.go`, change the `NewSingleFile` call to pass `watchFlag`:
```go
model, err = app.NewSingleFile(path, watchFlag)
```

Remove the `_ = watchFlag` line above.

- [ ] **Step 10: Run full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 11: Build and smoke-test**

```bash
go build -o stackreader .
./stackreader --version
```

Expected: prints `stackreader dev`

```bash
echo "# Hello\n\nworld" > /tmp/test.md
./stackreader /tmp/test.md
```

Expected: opens in single-file viewer mode (no sidebar). Press `q` to quit.

- [ ] **Step 12: Commit**

```bash
git add go.mod go.sum app/app.go app/app_test.go main.go
git commit -m "feat: add --watch flag — auto-reload on file save using fsnotify"
```

- [ ] **Step 13: Tag and push to trigger release**

Check the latest existing tag first:
```bash
git tag --sort=-v:refname | head -1
```

Increment the patch version. If the latest tag is `v0.3.1`, the new tag is `v0.3.2`:
```bash
git tag v0.3.2
git push origin master --tags
```

Expected: GitHub Actions workflow triggers, goreleaser builds `stackreader_*_darwin_amd64.tar.gz` etc. and publishes a GitHub release. Verify at `https://github.com/AlexTSPower/StackReader/releases`.

- [ ] **Step 14: Clean up local binary**

```bash
rm stackreader
```
