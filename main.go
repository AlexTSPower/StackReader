package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"terminal-markdown/app"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdv: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "mdv: %s is not a directory\n", root)
		os.Exit(1)
	}

	model, err := app.New(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdv: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdv: %v\n", err)
		os.Exit(1)
	}
}
