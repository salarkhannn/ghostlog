package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/salarkhannn/ghostlog/internal/analyzer"
	"github.com/salarkhannn/ghostlog/internal/git"
)

type exportBurst struct {
	Index        int      `json:"index"`
	Hashes       []string `json:"hashes"`
	StartTime    string   `json:"start_time"`
	LastTime     string   `json:"last_time"`
	DiffSHA256   string   `json:"diff_sha256"`
	FilesChanged int      `json:"files_changed"`
	LinesAdded   int      `json:"lines_added"`
	LinesRemoved int      `json:"lines_removed"`
	SecretLeaks  []string `json:"secret_leaks,omitempty"`
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	session := fs.String("session", "", "path to git repository")
	out := fs.String("out", "", "output manifest file")
	fs.Parse(args)

	if *session == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "Usage: ghostlog export -session <dir> -out <manifest.jsonl>")
		os.Exit(1)
	}

	bursts := extractBursts(*session)
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghostlog: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, b := range bursts {
		diff, _ := git.Show(*session, b.Hashes)
		hash := sha256.Sum256([]byte(diff))
		eb := exportBurst{
			Index:        b.ID,
			Hashes:       b.Hashes,
			StartTime:    b.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			LastTime:     b.LastTime.Format("2006-01-02T15:04:05Z07:00"),
			DiffSHA256:   hex.EncodeToString(hash[:]),
			FilesChanged: b.FilesChanged,
			LinesAdded:   b.LinesAdded,
			LinesRemoved: b.LinesRemoved,
			SecretLeaks:  b.SecretLeaks,
		}
		if err := enc.Encode(eb); err != nil {
			fmt.Fprintf(os.Stderr, "ghostlog: write error: %v\n", err)
			os.Exit(1)
		}
	}
}

func extractBursts(repoPath string) []analyzer.Burst {
	out, err := exec.Command("git", "-C", repoPath, "log", "--reverse", "--format=%H").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	az := analyzer.New(repoPath)
	var bursts []analyzer.Burst
	for _, h := range lines {
		if h == "" {
			continue
		}
		info, err := git.DiffTree(repoPath, h)
		if err == nil {
			bursts, _ = az.Add(info)
		}
	}
	return bursts
}
