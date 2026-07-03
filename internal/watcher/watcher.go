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
		lastHash: headHash(repoPath),
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
			
			// Dynamically watch newly created directories (like .git/logs)
			if ev.Has(fsnotify.Create) {
				if stat, err := os.Stat(ev.Name); err == nil && stat.IsDir() {
					w.fw.Add(ev.Name)
				}
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
		w.mu.Lock()
		last := w.lastHash
		w.mu.Unlock()

		var hashes []string
		if last == "" {
			// Fresh repo, get only the first commit ever
			out, _ := exec.Command("git", "-C", w.repoPath, "log", "--format=%H", "--reverse").Output()
			hashes = strings.Fields(string(out))
		} else {
			out, _ := exec.Command("git", "-C", w.repoPath, "log", last+"..HEAD", "--format=%H", "--reverse").Output()
			hashes = strings.Fields(string(out))
		}

		if len(hashes) == 0 {
			return
		}

		w.mu.Lock()
		w.lastHash = hashes[len(hashes)-1]
		w.mu.Unlock()

		for _, hash := range hashes {
			select {
			case w.ch <- CommitMsg{Hash: hash, RepoPath: w.repoPath}:
			case <-w.done:
				return
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
