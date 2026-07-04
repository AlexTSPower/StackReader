package app

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// fileItem is a single markdown file shown in the browser list.
type fileItem struct {
	path    string // absolute path
	display string // relative path shown in the list
}

func (f fileItem) FilterValue() string { return f.display }
func (f fileItem) Title() string       { return f.display }
func (f fileItem) Description() string { return "" }

// Browser is the sidebar file-browser model.
type Browser struct {
	root string
	list list.Model
}

// NewBrowser constructs a Browser rooted at root with the given dimensions.
func NewBrowser(root string, width, height int) (Browser, error) {
	items, err := findMarkdownFiles(root)
	if err != nil {
		return Browser{}, err
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	l := list.New(items, delegate, width, height)
	l.Title = "BROWSER"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.KeyMap = browserKeyMap()

	return Browser{root: root, list: l}, nil
}

// browserKeyMap returns a key map for the list that does not conflict with
// App-level bindings (b, i, q).
func browserKeyMap() list.KeyMap {
	return list.KeyMap{
		CursorUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PrevPage:             key.NewBinding(key.WithKeys("pgup")),
		NextPage:             key.NewBinding(key.WithKeys("pgdown")),
		GoToStart:            key.NewBinding(key.WithKeys("home", "g")),
		GoToEnd:              key.NewBinding(key.WithKeys("end", "G")),
		Filter:               key.NewBinding(key.WithKeys()),
		ClearFilter:          key.NewBinding(key.WithKeys()),
		CancelWhileFiltering: key.NewBinding(key.WithKeys()),
		AcceptWhileFiltering: key.NewBinding(key.WithKeys()),
		ShowFullHelp:         key.NewBinding(key.WithKeys()),
		CloseFullHelp:        key.NewBinding(key.WithKeys()),
		Quit:                 key.NewBinding(key.WithKeys()),
		ForceQuit:            key.NewBinding(key.WithKeys()),
	}
}

// Update handles messages for the Browser model.
func (b Browser) Update(msg tea.Msg) (Browser, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.list.SetSize(msg.Width, msg.Height)
		return b, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			if item, ok := b.list.SelectedItem().(fileItem); ok {
				path := item.path
				return b, func() tea.Msg { return FileSelectedMsg{Path: path} }
			}
			return b, nil
		}
	}
	var cmd tea.Cmd
	b.list, cmd = b.list.Update(msg)
	return b, cmd
}

// View renders the browser list or an empty-state message.
func (b Browser) View() string {
	if len(b.list.Items()) == 0 {
		return "\n  No markdown files found."
	}
	return b.list.View()
}

// findMarkdownFiles walks root recursively and returns list items for every
// .md and .mdx file found. Hidden directories (dot-prefixed) are skipped.
func findMarkdownFiles(root string) ([]list.Item, error) {
	var items []list.Item
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && isMarkdown(path) {
			rel, _ := filepath.Rel(root, path)
			items = append(items, fileItem{path: path, display: rel})
		}
		return nil
	})
	return items, err
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".mdx"
}
