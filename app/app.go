package app

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

const (
	minWidthForSidebar = 80
	sidebarRatio       = 0.20
	sidebarMin         = 18
	sidebarMax         = 30
)

type focusTarget int

const (
	focusBrowser focusTarget = iota
	focusViewer
)

// clearStatusMsg is sent by the auto-dismiss tick to clear the status bar.
type clearStatusMsg struct{}

// fileChangedMsg is sent by the fsnotify watcher when the open file changes.
type fileChangedMsg struct{}

// editorCandidates is the fallback order when $EDITOR is not set.
// Declared as a var so tests can override it.
var editorCandidates = []string{"nvim", "vim", "nano"}

// findEditor returns the editor binary to launch: $EDITOR if set, else the
// first installed candidate from editorCandidates.
func findEditor() (string, error) {
	if e := os.Getenv("EDITOR"); e != "" {
		return e, nil
	}
	for _, name := range editorCandidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no editor found; install nvim, vim, or nano, or set $EDITOR")
}

// App is the root bubbletea model. It owns layout and routes messages.
type App struct {
	browser     Browser
	viewer      Viewer
	width       int
	height      int
	showSidebar bool
	focus       focusTarget
	currentFile string
	statusMsg   string
	singleFile  string           // non-empty when started with a file argument
	watcher     *fsnotify.Watcher // non-nil when --watch is active
}

// New constructs the root App model rooted at root.
func New(root string) (App, error) {
	browser, err := NewBrowser(root, sidebarMin, 20)
	if err != nil {
		return App{}, err
	}
	return App{
		browser:     browser,
		viewer:      NewViewer(80, 20),
		showSidebar: true,
		focus:       focusBrowser,
	}, nil
}

// NewSingleFile constructs an App in single-file mode: no browser, viewer fills
// the full terminal. When watch is true, an fsnotify watcher is created and
// the app will auto-reload the file whenever it changes on disk.
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
		// Watch the parent directory: atomic-save editors rename a temp file over the
		// target, which would destroy a file-level watch. Directory-level watching
		// survives the rename.
		if err := w.Add(filepath.Dir(absPath)); err != nil {
			w.Close()
			return App{}, err
		}
		a.watcher = w
	}
	return a, nil
}

func (a App) Init() tea.Cmd {
	if a.singleFile == "" {
		return nil
	}
	path := a.singleFile
	cmds := []tea.Cmd{
		func() tea.Msg { return FileSelectedMsg{Path: path} },
	}
	if a.watcher != nil {
		cmds = append(cmds, watchFileCmd(a.watcher, a.singleFile))
	}
	return tea.Batch(cmds...)
}

// setStatus sets a status bar message and schedules auto-clear after 3 s.
func (a App) setStatus(msg string) (App, tea.Cmd) {
	a.statusMsg = msg
	return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

// watchFileCmd returns a Cmd that blocks until a Write, Create, or Rename event
// fires on the target file, then returns fileChangedMsg. Must be re-issued after
// each event to keep the watch loop running. targetPath must be an absolute,
// cleaned path. The watcher watches the parent directory so atomic-save editors
// (vim, nvim) that rename a temp file over the target don't lose the watch.
func watchFileCmd(w *fsnotify.Watcher, targetPath string) tea.Cmd {
	return func() tea.Msg {
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return nil
				}
				// Directory watching picks up all files; filter to ours only.
				if filepath.Clean(event.Name) != targetPath {
					continue
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
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

// Update handles all messages for the App.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a = a.applyLayout()
		return a, nil

	case FileSelectedMsg:
		a.currentFile = msg.Path
		a.focus = focusViewer
		var cmd tea.Cmd
		a.viewer, cmd = a.viewer.Update(msg)
		return a, cmd

	case clearStatusMsg:
		a.statusMsg = ""
		return a, nil

	case fileChangedMsg:
		a.currentFile = a.singleFile // guard against startup race with FileSelectedMsg
		// Re-read the file and re-render, then re-arm the watcher.
		var cmd tea.Cmd
		a.viewer, cmd = a.viewer.Update(FileSelectedMsg{Path: a.singleFile})
		var wCmd tea.Cmd
		if a.watcher != nil {
			wCmd = watchFileCmd(a.watcher, a.singleFile)
		}
		return a, tea.Batch(cmd, wCmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "b":
			if a.singleFile != "" {
				return a, nil
			}
			a.showSidebar = !a.showSidebar
			a = a.applyLayout()
			return a, nil
		case "tab":
			if a.sidebarWidth() > 0 {
				if a.focus == focusBrowser {
					a.focus = focusViewer
				} else {
					a.focus = focusBrowser
				}
			}
			return a, nil
		case "i":
			if a.currentFile == "" {
				return a, nil
			}
			editor, err := findEditor()
			if err != nil {
				return a.setStatus("Error: " + err.Error())
			}
			a.statusMsg = ""
			file := a.currentFile
			cmd := exec.Command(editor, file)
			return a, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return FileSelectedMsg{Path: file}
			})
		default:
			if a.focus == focusBrowser && a.sidebarWidth() > 0 {
				var cmd tea.Cmd
				a.browser, cmd = a.browser.Update(msg)
				return a, cmd
			}
			var cmd tea.Cmd
			a.viewer, cmd = a.viewer.Update(msg)
			return a, cmd
		}
	}
	return a, nil
}

// View renders the full TUI: title bar, content area, status bar.
func (a App) View() string {
	sw := a.sidebarWidth()
	vw := a.width - sw
	if sw > 0 {
		vw-- // separator column
	}
	contentH := a.height - 2 // title + status bars

	title := "stackreader"
	if a.currentFile != "" {
		title = "stackreader — " + filepath.Base(a.currentFile)
	}
	titleBar := lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1).
		Render(title)

	help := "[b] sidebar  [tab] focus  [i] edit  [q] quit  [↑↓/jk] scroll"
	if a.statusMsg != "" {
		help = a.statusMsg
	} else if sw > 0 {
		// Show which panel currently has focus.
		focusLabel := "browser"
		if a.focus == focusViewer {
			focusLabel = "viewer"
		}
		help += "  │ " + focusLabel
	}
	statusBar := lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("250")).
		Padding(0, 1).
		Render(help)

	var body string
	if sw > 0 {
		// Highlight the sidebar border when the browser has focus.
		borderColor := lipgloss.Color("238")
		if a.focus == focusBrowser {
			borderColor = lipgloss.Color("62")
		}
		sidebar := lipgloss.NewStyle().
			Width(sw).
			Height(contentH).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			Render(a.browser.View())
		viewer := lipgloss.NewStyle().
			Width(vw).
			Height(contentH).
			Render(a.viewer.View())
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, viewer)
	} else {
		body = lipgloss.NewStyle().
			Width(a.width).
			Height(contentH).
			Render(a.viewer.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, body, statusBar)
}

// sidebarWidth returns the current sidebar width in columns (0 when hidden).
func (a App) sidebarWidth() int {
	if !a.showSidebar || a.width < minWidthForSidebar {
		return 0
	}
	w := int(float64(a.width) * sidebarRatio)
	if w < sidebarMin {
		return sidebarMin
	}
	if w > sidebarMax {
		return sidebarMax
	}
	return w
}

// applyLayout recalculates and forwards dimensions to child models via their
// Update methods (rather than poking their internals directly).
func (a App) applyLayout() App {
	sw := a.sidebarWidth()
	vw := a.width - sw
	if sw > 0 {
		vw--
	}
	contentH := a.height - 2

	var cmd tea.Cmd
	// Only resize the browser when the sidebar is visible; calling
	// SetSize(0, h) can cause display issues in the list component.
	if sw > 0 {
		a.browser, cmd = a.browser.Update(tea.WindowSizeMsg{Width: sw, Height: contentH})
		_ = cmd
	}
	a.viewer, cmd = a.viewer.Update(tea.WindowSizeMsg{Width: vw, Height: contentH})
	_ = cmd
	return a
}
