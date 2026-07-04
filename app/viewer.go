package app

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// Viewer renders a single markdown file in a scrollable viewport.
type Viewer struct {
	viewport   viewport.Model
	rawContent string // stored for re-render on resize
	ready      bool
}

// NewViewer constructs a Viewer with the given dimensions.
func NewViewer(width, height int) Viewer {
	vp := viewport.New(width, height)
	vp.SetContent("\n  Select a file to preview.")
	return Viewer{viewport: vp}
}

// renderContent renders the stored raw markdown using glamour, wrapping at the
// current viewport width. It falls back to the raw content if glamour fails.
func (v Viewer) renderContent() string {
	if v.rawContent == "" {
		return ""
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(v.viewport.Width),
	)
	if err != nil {
		return v.rawContent
	}
	out, err := r.Render(v.rawContent)
	if err != nil {
		return v.rawContent
	}
	return out
}

// Update handles messages for the Viewer model.
func (v Viewer) Update(msg tea.Msg) (Viewer, tea.Cmd) {
	switch msg := msg.(type) {
	case FileSelectedMsg:
		content, err := os.ReadFile(msg.Path)
		if err != nil {
			v.rawContent = ""
			v.viewport.SetContent(fmt.Sprintf("\n  Error reading file: %v", err))
			v.ready = true
			return v, nil
		}
		v.rawContent = string(content)
		v.viewport.SetContent(v.renderContent())
		v.viewport.GotoTop()
		v.ready = true
		return v, nil

	case tea.WindowSizeMsg:
		v.viewport.Width = msg.Width
		v.viewport.Height = msg.Height
		if v.ready {
			v.viewport.SetContent(v.renderContent())
		}
		return v, nil
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the viewport content.
func (v Viewer) View() string {
	return v.viewport.View()
}
