package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFindMarkdownFiles_FiltersCorrectly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(dir, "guide.mdx"), []byte("# Guide"), 0644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "docs", "api.md"), []byte("# API"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "hidden.md"), []byte("# Hidden"), 0644)

	items, err := findMarkdownFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Expects: README.md, guide.mdx, docs/api.md (not notes.txt, not .git/hidden.md)
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}

func TestBrowser_EnterEmitsFileSelectedMsg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	os.WriteFile(path, []byte("# Hello"), 0644)

	b, err := NewBrowser(dir, 30, 20)
	if err != nil {
		t.Fatal(err)
	}

	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}

	msg := cmd()
	sel, ok := msg.(FileSelectedMsg)
	if !ok {
		t.Fatalf("expected FileSelectedMsg, got %T", msg)
	}
	if sel.Path != path {
		t.Errorf("got path %q, want %q", sel.Path, path)
	}
}

func TestBrowser_EmptyDir_ViewShowsEmptyState(t *testing.T) {
	dir := t.TempDir()
	b, err := NewBrowser(dir, 30, 20)
	if err != nil {
		t.Fatal(err)
	}
	view := b.View()
	if view == "" {
		t.Error("expected non-empty view for empty directory")
	}
}
