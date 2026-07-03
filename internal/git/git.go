package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type FileDiff struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
	Bytes        int
}

type CommitInfo struct {
	Hash  string
	Short string
	Diffs []FileDiff
}

var cache sync.Map

func DiffTree(repoPath, hash string) (*CommitInfo, error) {
	if v, ok := cache.Load(hash); ok {
		return v.(*CommitInfo), nil
	}

	out, err := exec.Command("git", "-C", repoPath, "diff-tree", "--no-commit-id", "-p", "-r", hash).Output()
	if err != nil {
		return nil, fmt.Errorf("diff-tree %s: %w", shortHash(hash), err)
	}

	info := &CommitInfo{
		Hash:  hash,
		Short: shortHash(hash),
		Diffs: parseDiff(out),
	}
	cache.Store(hash, info)
	return info, nil
}

func Show(repoPath string, hashes []string) (string, error) {
	args := append([]string{"-C", repoPath, "show", "--color=always", "--stat", "--patch"}, hashes...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git show: %w", err)
	}
	return string(out), nil
}

func shortHash(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}

func parseDiff(data []byte) []FileDiff {
	var diffs []FileDiff
	var cur *FileDiff

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if cur != nil {
				diffs = append(diffs, *cur)
			}
			cur = &FileDiff{}
		case strings.HasPrefix(line, "+++ b/") && cur != nil:
			cur.Path = strings.TrimPrefix(line, "+++ b/")
		case cur != nil && strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			cur.LinesAdded++
			cur.Bytes += len(line) - 1
		case cur != nil && strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			cur.LinesRemoved++
		}
	}
	if cur != nil {
		diffs = append(diffs, *cur)
	}
	return diffs
}
