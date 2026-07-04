package app

import (
	"os"
	"os/exec"
	"path/filepath"

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

func (a App) Init() tea.Cmd { return nil }

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

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "b":
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
			editor := os.Getenv("EDITOR")
			if editor == "" {
				a.statusMsg = "Error: $EDITOR is not set"
				return a, nil
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

	title := "mdv"
	if a.currentFile != "" {
		title = "mdv — " + filepath.Base(a.currentFile)
	}
	titleBar := lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1).
		Render(title)

	help := "[b] browser  [tab] focus  [i] edit  [q] quit  [↑↓/jk] scroll"
	if a.statusMsg != "" {
		help = a.statusMsg
	}
	statusBar := lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("250")).
		Padding(0, 1).
		Render(help)

	var body string
	if sw > 0 {
		sidebar := lipgloss.NewStyle().
			Width(sw).
			Height(contentH).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
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
