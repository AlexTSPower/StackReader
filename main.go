package main

import (
	"fmt"
	"os"
	"path/filepath"
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
		model, err = app.NewSingleFile(path, watchFlag)
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
