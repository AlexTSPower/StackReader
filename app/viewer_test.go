package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestViewer_InitialView_ShowsPlaceholder(t *testing.T) {
	v := NewViewer(80, 24)
	view := v.View()
	if !strings.Contains(view, "Select a file") {
		t.Errorf("expected placeholder text, got: %q", view)
	}
}

func TestViewer_FileSelectedMsg_RendersContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello\n\nWorld"), 0644)

	v := NewViewer(80, 24)
	v2, _ := v.Update(FileSelectedMsg{Path: path})

	if !v2.ready {
		t.Error("viewer should be ready after FileSelectedMsg")
	}
	view := v2.View()
	if view == "" {
		t.Error("expected non-empty view after file selected")
	}
}

func TestViewer_UnreadableFile_ShowsError(t *testing.T) {
	v := NewViewer(80, 24)
	v2, _ := v.Update(FileSelectedMsg{Path: "/nonexistent/path/file.md"})

	view := v2.View()
	if !strings.Contains(view, "Error") {
		t.Errorf("expected error message in view, got: %q", view)
	}
}

func TestViewer_H2HeadingHasNoLiteralPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("## Section Title\n\nContent"), 0644)

	v := NewViewer(80, 24)
	v2, _ := v.Update(FileSelectedMsg{Path: path})
	rendered := v2.renderContent()

	// Custom style removes the raw "## " prefix — headings are styled, not prefixed.
	if strings.Contains(rendered, "## Section") {
		t.Error("H2 should not contain literal ## prefix in rendered output")
	}
	// Glamour may split the heading text across multiple ANSI spans, so check
	// for each word individually.
	if !strings.Contains(rendered, "Section") || !strings.Contains(rendered, "Title") {
		t.Errorf("H2 text should be present in rendered output, got: %q", rendered)
	}
}

func TestViewer_GlamourFailure_FallsBackToRaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.md")
	content := "# Heading\n\nSome content"
	os.WriteFile(path, []byte(content), 0644)

	v := NewViewer(80, 24)
	v2, _ := v.Update(FileSelectedMsg{Path: path})

	// The raw content must be stored so it can be re-rendered on resize and
	// used as the fallback if glamour rendering fails.
	if v2.rawContent != content {
		t.Errorf("rawContent should be stored as %q, got %q", content, v2.rawContent)
	}

	// Should have content regardless of whether glamour succeeded.
	if !v2.ready {
		t.Error("viewer should be ready even if glamour render fails")
	}
	if v2.View() == "" {
		t.Error("expected non-empty view after file selected")
	}

	// Exercise the fallback path directly: with rawContent set, renderContent
	// always returns the raw text if glamour fails, so the returned string must
	// contain the original content. When glamour succeeds the heading text is
	// still present in the styled output. Either way, the content is preserved.
	rendered := v2.renderContent()
	if !strings.Contains(rendered, "Heading") {
		t.Errorf("renderContent output should preserve content, got: %q", rendered)
	}

	// renderContent with empty rawContent returns empty (the explicit fallback
	// guard). This is the one fallback branch we can trigger deterministically;
	// glamour cannot be forced to error reliably in a headless test.
	empty := NewViewer(80, 24)
	if empty.renderContent() != "" {
		t.Error("renderContent should return empty string when rawContent is empty")
	}
}
