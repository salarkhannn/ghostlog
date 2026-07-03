package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/salarkhannn/ghostlog/internal/tui"
	"github.com/salarkhannn/ghostlog/internal/watcher"
)

func main() {
	repoPath := flag.String("repo", ".", "path to git repository to watch")
	flag.Parse()

	abs, err := filepath.Abs(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}

	ch := make(chan watcher.CommitMsg, 32)
	w, err := watcher.New(abs, ch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}
	w.Start()
	defer w.Stop()

	p := tea.NewProgram(
		tui.New(abs, ch),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}
}
