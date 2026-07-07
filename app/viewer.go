package app

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// mdvStyle is a custom glamour style based on the dark theme with improved
// heading hierarchy (no raw ## prefix, colour+underline per level) and better
// code-block contrast.
const mdvStyle = `{
  "document": { "margin": 2, "block_suffix": "\n" },
  "block_quote": { "indent": 1, "indent_token": "│ ", "color": "252", "italic": true },
  "list": { "level_indent": 2 },
  "heading": { "block_suffix": "\n", "bold": true },
  "h1": {
    "prefix": " ", "suffix": " ",
    "background_color": "57", "color": "230", "bold": true
  },
  "h2": { "color": "14",  "bold": true, "underline": true, "block_prefix": "\n" },
  "h3": { "color": "12",  "bold": true },
  "h4": { "color": "75",  "bold": true },
  "h5": { "color": "33" },
  "h6": { "color": "33",  "italic": true },
  "strikethrough": { "crossed_out": true },
  "emph":   { "italic": true, "color": "252" },
  "strong": { "bold": true,   "color": "252" },
  "hr": {
    "color": "240",
    "format": "\n------------------------------------------------------------------------\n"
  },
  "item":        { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "✓ ", "unticked": "✗ " },
  "link":       { "color": "33", "underline": true },
  "link_text":  { "color": "39", "bold": true },
  "image":      { "color": "33", "underline": true },
  "image_text": { "color": "243", "format": "Image: {{.text}}" },
  "code":       { "background_color": "236", "color": "203" },
  "code_block": {
    "background_color": "235",
    "color": "244",
    "padding": 1,
    "margin": 2,
    "chroma": {
      "text":                   { "color": "#C4C4C4" },
      "error":                  { "color": "#F1F1F1", "background_color": "#F05B5B" },
      "comment":                { "color": "#888888" },
      "comment_preproc":        { "color": "#FF875F" },
      "comment_special":        { "color": "#FF5F87" },
      "keyword":                { "bold": true },
      "keyword_declaration":    { "color": "#FF875F" },
      "keyword_namespace":      { "color": "#FF875F" },
      "keyword_type":           { "color": "#6E6ED8" },
      "operator":               { "color": "#EF8080" },
      "punctuation":            { "color": "#E8E8A8" },
      "name":                   { "color": "#C4C4C4" },
      "name_builtin":           { "color": "#FF875F" },
      "name_tag":               { "color": "#B083EA" },
      "name_attribute":         { "color": "#7EC4CF" },
      "name_class":             { "bold": true, "color": "#FF875F" },
      "name_constant":          { "color": "#FF875F" },
      "name_decorator":         { "color": "#FF875F" },
      "name_exception":         { "color": "#FF875F" },
      "name_function":          { "color": "#FF875F" },
      "name_other":             { "color": "#FF875F" },
      "name_label":             { "color": "#FF875F" },
      "literal_number":         { "color": "#6EEFC0" },
      "literal_string":         { "color": "#6EEFC0" },
      "literal_string_escape":  { "bold": true, "color": "#FF875F" },
      "generic_heading":        { "bold": true },
      "generic_subheading":     { "color": "#777777" },
      "generic_deleted":        { "color": "#F92672" },
      "generic_emph":           { "italic": true },
      "generic_inserted":       { "color": "#A6E22E" },
      "generic_strong":         { "bold": true },
      "generic_output":         { "color": "#777777" },
      "background":             { "background_color": "235" }
    }
  },
  "table": {
    "center_separator": "┼",
    "column_separator": "│",
    "row_separator":    "─"
  },
  "definition_description": { "block_prefix": "\n→ " }
}`

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
		glamour.WithStylesFromJSONBytes([]byte(mdvStyle)),
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
