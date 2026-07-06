package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestApp(t *testing.T) App {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0644)
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Give it dimensions so layout calculations work
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return model.(App)
}

func TestApp_QuitKey(t *testing.T) {
	a := newTestApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestApp_SidebarToggle(t *testing.T) {
	a := newTestApp(t)
	if !a.showSidebar {
		t.Fatal("sidebar should be visible by default")
	}
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	a2 := model.(App)
	if a2.showSidebar {
		t.Error("sidebar should be hidden after pressing b")
	}
	model, _ = a2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	a3 := model.(App)
	if !a3.showSidebar {
		t.Error("sidebar should be visible after pressing b again")
	}
}

func TestApp_EditKey_NoopWhenNoFileOpen(t *testing.T) {
	a := newTestApp(t)
	// No file selected yet; currentFile is empty
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if cmd != nil {
		t.Error("expected nil command when no file is open")
	}
}

func TestApp_EditKey_ErrorWhenEditorNotFound(t *testing.T) {
	a := newTestApp(t)
	a.currentFile = "/some/file.md"
	t.Setenv("EDITOR", "")

	// Override candidates so no real editor is found.
	orig := editorCandidates
	editorCandidates = []string{"__no_such_editor_1234__"}
	defer func() { editorCandidates = orig }()

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	a2 := model.(App)
	if a2.statusMsg == "" {
		t.Error("expected an error status message when no editor found")
	}
}

func TestApp_FileSelectedMsg_FocusesViewer(t *testing.T) {
	a := newTestApp(t)
	// Focus starts on browser
	if a.focus != focusBrowser {
		t.Fatal("focus should start on browser")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Test"), 0644)

	model, _ := a.Update(FileSelectedMsg{Path: path})
	a2 := model.(App)
	if a2.focus != focusViewer {
		t.Error("focus should switch to viewer after FileSelectedMsg")
	}
	if a2.currentFile != path {
		t.Errorf("currentFile should be %q, got %q", path, a2.currentFile)
	}
}

// --- editor fallback ---

func TestFindEditor_UsesEditorEnv(t *testing.T) {
	t.Setenv("EDITOR", "/usr/bin/env")
	path, err := findEditor()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/usr/bin/env" {
		t.Errorf("got %q, want /usr/bin/env", path)
	}
}

func TestFindEditor_FallsBackToCandidate(t *testing.T) {
	t.Setenv("EDITOR", "")

	// Use /bin/sh as a guaranteed-present candidate.
	orig := editorCandidates
	editorCandidates = []string{"__missing__", "sh"}
	defer func() { editorCandidates = orig }()

	path, err := findEditor()
	if err != nil {
		t.Fatalf("expected a fallback editor, got error: %v", err)
	}
	if path == "" {
		t.Error("expected a non-empty path from fallback")
	}
}

func TestFindEditor_ErrorWhenNoneFound(t *testing.T) {
	t.Setenv("EDITOR", "")

	orig := editorCandidates
	editorCandidates = []string{"__no_such_editor_a__", "__no_such_editor_b__"}
	defer func() { editorCandidates = orig }()

	_, err := findEditor()
	if err == nil {
		t.Error("expected error when no editor is found")
	}
}

// --- status message auto-clear ---

func TestApp_StatusMsg_ClearsOnClearMsg(t *testing.T) {
	a := newTestApp(t)
	a.statusMsg = "some error"

	model, _ := a.Update(clearStatusMsg{})
	a2 := model.(App)
	if a2.statusMsg != "" {
		t.Errorf("statusMsg should be cleared by clearStatusMsg, got %q", a2.statusMsg)
	}
}

func TestApp_EditKey_StatusMsgSchedulesClear(t *testing.T) {
	a := newTestApp(t)
	a.currentFile = "/some/file.md"
	t.Setenv("EDITOR", "")

	orig := editorCandidates
	editorCandidates = []string{"__no_such_editor_1234__"}
	defer func() { editorCandidates = orig }()

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	// A tick command must be returned so the status bar eventually clears.
	if cmd == nil {
		t.Error("expected a tick command to auto-clear the status message")
	}
}

// --- tab focus indicator ---

func TestApp_TabKey_CyclesFocus(t *testing.T) {
	a := newTestApp(t)
	if a.focus != focusBrowser {
		t.Fatal("focus should start on browser")
	}

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a2 := model.(App)
	if a2.focus != focusViewer {
		t.Error("tab should move focus to viewer")
	}

	model, _ = a2.Update(tea.KeyMsg{Type: tea.KeyTab})
	a3 := model.(App)
	if a3.focus != focusBrowser {
		t.Error("tab should cycle back to browser")
	}
}

func TestApp_TabFocus_ViewReflectsFocus(t *testing.T) {
	a := newTestApp(t)
	viewBrowser := a.View()

	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a2 := model.(App)
	viewViewer := a2.View()

	// The border colour and status bar label change, so the views must differ.
	if viewBrowser == viewViewer {
		t.Error("View() should differ when focus changes (border colour and status label)")
	}

	// Status bar should indicate the focused panel name.
	if !strings.Contains(viewBrowser, "browser") {
		t.Error("status bar should say 'browser' when browser is focused")
	}
	if !strings.Contains(viewViewer, "viewer") {
		t.Error("status bar should say 'viewer' when viewer is focused")
	}
}

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

	// Glamour splits heading text across multiple ANSI spans so
	// strings.Contains(view, "Version 2") is unreliable; rawContent stores the
	// unprocessed file bytes and is the authoritative signal that the file was
	// re-read from disk.
	if !strings.Contains(a4.viewer.rawContent, "Version 2") {
		t.Errorf("viewer should show updated content after fileChangedMsg, got rawContent: %q", a4.viewer.rawContent)
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
