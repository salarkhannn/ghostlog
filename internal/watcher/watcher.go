package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDuration = 500 * time.Millisecond

type CommitEvent struct {
	RepoPath string
}

type Watcher struct {
	w        *fsnotify.Watcher
	repoPath string
	Events   chan CommitEvent
	Errors   chan error

	mu      sync.Mutex
	timer   *time.Timer
	stopped chan struct{}
}

func New(repoPath string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(filepath.Join(gitDir, "HEAD")); err != nil {
		fw.Close()
		return nil, fmt.Errorf("%s is not a git repository", repoPath)
	}

	for _, sub := range []string{"refs", "logs"} {
		if err := addRecursive(fw, filepath.Join(gitDir, sub)); err != nil {
			fw.Close()
			return nil, err
		}
	}

	if err := fw.Add(gitDir); err != nil {
		fw.Close()
		return nil, err
	}

	return &Watcher{
		w:        fw,
		repoPath: repoPath,
		Events:   make(chan CommitEvent, 16),
		Errors:   make(chan error, 8),
		stopped:  make(chan struct{}),
	}, nil
}

func (w *Watcher) Start() {
	go w.loop()
}

func (w *Watcher) Stop() {
	close(w.stopped)
	w.w.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.stopped:
			return
		case event, ok := <-w.w.Events:
			if !ok {
				return
			}
			if isRelevant(event) {
				w.debounce()
			}
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			select {
			case w.Errors <- err:
			default:
			}
		}
	}
}

func (w *Watcher) debounce() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Reset(debounceDuration)
		return
	}

	w.timer = time.AfterFunc(debounceDuration, func() {
		w.mu.Lock()
		w.timer = nil
		w.mu.Unlock()

		select {
		case w.Events <- CommitEvent{RepoPath: w.repoPath}:
		case <-w.stopped:
		}
	})
}

func isRelevant(e fsnotify.Event) bool {
	if !e.Has(fsnotify.Write) && !e.Has(fsnotify.Create) {
		return false
	}
	base := filepath.Base(e.Name)
	switch base {
	case "HEAD", "ORIG_HEAD", "MERGE_HEAD", "COMMIT_EDITMSG":
		return true
	}
	if strings.Contains(e.Name, ".git/logs/") || strings.Contains(e.Name, ".git/refs/") {
		return true
	}
	return false
}

func addRecursive(fw *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return fw.Add(path)
		}
		return nil
	})
}
