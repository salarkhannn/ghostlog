package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/salarkhannn/ghostlog/internal/tui"
	"github.com/salarkhannn/ghostlog/internal/watcher"
)

func main() {
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		switch os.Args[1] {
		case "export":
			runExport(os.Args[2:])
			return
		}
	}

	repoPath := flag.String("repo", ".", "path to git repository to watch")
	flag.Parse()

	abs, err := filepath.Abs(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(filepath.Join(abs, ".git", "HEAD")); err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %s is not a git repository\n", abs)
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
