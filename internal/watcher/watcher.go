package watcher

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

type CommitMsg struct {
	Hash     string
	RepoPath string
}

type Watcher struct {
	fw       *fsnotify.Watcher
	repoPath string
	ch       chan<- CommitMsg
	done     chan struct{}

	mu       sync.Mutex
	lastHash string
	timer    *time.Timer
}

func New(repoPath string, ch chan<- CommitMsg) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	gitDir := filepath.Join(repoPath, ".git")
	for _, sub := range []string{"", "logs", "logs/refs", "logs/refs/heads", "refs", "refs/heads"} {
		_ = fw.Add(filepath.Join(gitDir, sub))
	}

	return &Watcher{
		fw:       fw,
		repoPath: repoPath,
		ch:       ch,
		done:     make(chan struct{}),
	}, nil
}

func (w *Watcher) Start() { go w.loop() }

func (w *Watcher) Stop() {
	close(w.done)
	w.fw.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			if isRelevant(ev) {
				w.arm()
			}
		case _, ok := <-w.fw.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) arm() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(500*time.Millisecond, func() {
		hash := headHash(w.repoPath)
		if hash == "" {
			return
		}
		w.mu.Lock()
		dup := hash == w.lastHash
		if !dup {
			w.lastHash = hash
		}
		w.mu.Unlock()

		if !dup {
			select {
			case w.ch <- CommitMsg{Hash: hash, RepoPath: w.repoPath}:
			case <-w.done:
			}
		}
	})
}

func WaitForCommit(ch <-chan CommitMsg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func isRelevant(e fsnotify.Event) bool {
	if !e.Has(fsnotify.Write) && !e.Has(fsnotify.Create) {
		return false
	}
	base := filepath.Base(e.Name)
	switch base {
	case "HEAD", "COMMIT_EDITMSG", "ORIG_HEAD":
		return true
	}
	return strings.Contains(e.Name, ".git/logs/") || strings.Contains(e.Name, ".git/refs/")
}

func headHash(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%H").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
