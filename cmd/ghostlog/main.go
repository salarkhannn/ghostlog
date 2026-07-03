package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/salarkhannn/ghostlog/internal/analyzer"
	"github.com/salarkhannn/ghostlog/internal/git"
	glog "github.com/salarkhannn/ghostlog/internal/log"
	"github.com/salarkhannn/ghostlog/internal/watcher"
)

func main() {
	repoPath := flag.String("repo", ".", "path to the git repository to watch")
	outputPath := flag.String("out", "", "path to the output JSONL log file (default: <repo>/.ghostlog.jsonl)")
	flag.Parse()

	absRepo, err := filepath.Abs(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: invalid repo path: %v\n", err)
		os.Exit(1)
	}

	out := *outputPath
	if out == "" {
		out = filepath.Join(absRepo, ".ghostlog.jsonl")
	}

	mgr, err := glog.NewManager(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}

	w, err := watcher.New(absRepo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "ghostlog: watching %s...\n", absRepo)

	w.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var lastHash string

	for {
		select {
		case <-sigs:
			fmt.Fprintf(os.Stderr, "ghostlog: caught interrupt, saving log...\n")
			w.Stop()
			mgr.Close()
			return

		case ev := <-w.Events:
			hash, err := git.HeadCommit(ev.RepoPath)
			if err != nil || hash == lastHash {
				continue
			}
			lastHash = hash

			info, err := git.DiffTree(ev.RepoPath, hash)
			if err != nil {
				continue
			}

			analyses := make([]analyzer.FileAnalysis, len(info.Diffs))
			for i, d := range info.Diffs {
				analyses[i] = analyzer.Analyze(d)
			}

			record := glog.NewRecord(info.Hash, analyses)
			if err := mgr.Write(record); err != nil {
				fmt.Fprintf(os.Stderr, "ghostlog: write error: %v\n", err)
			}

		case err := <-w.Errors:
			fmt.Fprintf(os.Stderr, "ghostlog: watcher error: %v\n", err)
		}
	}
}
