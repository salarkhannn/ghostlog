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
		case "check":
			runCheck(os.Args[2:])
			return
		}
	}

	repoPath := flag.String("repo", ".", "path to git repository to watch")
	iconsFlag := flag.String("icons", "", "icon mode: emoji, nerd, none")
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

	iconModeStr := *iconsFlag
	if iconModeStr == "" {
		iconModeStr = os.Getenv("GHOSTLOG_ICONS")
	}
	
	var iconMode tui.IconMode
	switch strings.ToLower(iconModeStr) {
	case "nerd":
		iconMode = tui.IconModeNerdFont
	case "emoji":
		iconMode = tui.IconModeEmoji
	case "none", "ascii":
		iconMode = tui.IconModeAscii
	case "auto", "":
		iconMode = detectIconMode()
	default:
		iconMode = tui.IconModeNerdFont
	}

	p := tea.NewProgram(
		tui.New(abs, ch, iconMode),
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)
	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}

	if tuiModel, ok := m.(tui.Model); ok {
		fmt.Fprintln(os.Stderr, tuiModel.Verdict())
	}
	os.Stdout.WriteString("\x1b[?7h")
}

// detectIconMode applies a heuristic to determine the best default icon mode
// based on the user's terminal emulator capabilities.
func detectIconMode() tui.IconMode {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// Raw TTY consoles usually have very limited font support
	if term == "linux" {
		return tui.IconModeAscii
	}

	// Apple Terminal has notoriously rigid font fallback
	if termProgram == "Apple_Terminal" {
		return tui.IconModeEmoji
	}

	// Terminals known to have excellent built-in symbol mapping, 
	// bundled Nerd Fonts, or power-user demographics that typically install them.
	if termProgram == "ghostty" || termProgram == "WezTerm" || termProgram == "iTerm.app" || term == "xterm-kitty" {
		return tui.IconModeNerdFont
	}

	// For generic terminals (gnome-terminal, xterm, basic WSL), fallback to universal emojis
	// to guarantee the UI doesn't look broken out-of-the-box.
	return tui.IconModeEmoji
}
