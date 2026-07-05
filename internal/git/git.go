package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/salarkhannn/ghostlog/internal/secrets"
)

type FileDiff struct {
	Path        string
	LinesAdded  int
	LinesRemoved int
	Bytes       int
	HasConflict bool
	ChangedLines [][2]int
	SecretLeaks []string
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

	out, err := exec.Command("git", "-C", repoPath, "diff-tree", "--root", "--no-commit-id", "-p", "-r", hash).Output()
	if err != nil {
		return nil, fmt.Errorf("diff-tree %s: %w", short(hash), err)
	}

	info := &CommitInfo{Hash: hash, Short: short(hash), Diffs: parseDiff(out)}
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

func ShowFile(repoPath, hash, path string) ([]byte, error) {
	out, err := exec.Command("git", "-C", repoPath, "show", fmt.Sprintf("%s:%s", hash, path)).Output()
	if err != nil {
		return nil, nil // Assume file doesn't exist at this hash
	}
	return out, nil
}

func DiffUncommitted(repoPath string) (*CommitInfo, error) {
	out, err := exec.Command("git", "-C", repoPath, "diff", "HEAD").Output()
	if err != nil {
		out, err = exec.Command("git", "-C", repoPath, "diff").Output()
		if err != nil {
			return nil, err
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return &CommitInfo{Hash: "UNCOMMITTED", Short: "UNCOMMITTED", Diffs: parseDiff(out)}, nil
}

func ShowUncommitted(repoPath string) (string, error) {
	out, err := exec.Command("git", "-C", repoPath, "diff", "HEAD", "--color=always", "--stat", "--patch").Output()
	if err != nil {
		out, err = exec.Command("git", "-C", repoPath, "diff", "--color=always", "--stat", "--patch").Output()
		if err != nil {
			return "", err
		}
	}
	return string(out), nil
}

func ListFiles(repoPath, hash, dir string) ([]string, error) {
	if dir == "." {
		dir = ""
	}
	out, err := exec.Command("git", "-C", repoPath, "ls-tree", "--name-only", "-r", hash, dir).Output()
	if err != nil {
		return nil, err
	}
	var files []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		files = append(files, scanner.Text())
	}
	return files, nil
}


func short(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}

func parseDiff(data []byte) []FileDiff {
	var diffs []FileDiff
	var cur *FileDiff
	var curLine int

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if cur != nil {
				diffs = append(diffs, *cur)
			}
			cur = &FileDiff{}
			curLine = 0
		case strings.HasPrefix(line, "+++ b/") && cur != nil:
			cur.Path = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "@@ ") && cur != nil:
			// parse @@ -start,len +start,len @@
			parts := strings.Split(line, " ")
			if len(parts) >= 3 && strings.HasPrefix(parts[2], "+") {
				plus := strings.TrimPrefix(parts[2], "+")
				sp := strings.Split(plus, ",")
				fmt.Sscanf(sp[0], "%d", &curLine)
				// track block if needed, but we can just track added lines as they appear
			}
		case cur != nil && strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			cur.LinesAdded++
			cur.Bytes += len(line) - 1
			if curLine > 0 {
				cur.ChangedLines = append(cur.ChangedLines, [2]int{curLine, curLine})
				content := strings.TrimPrefix(line, "+")
				if ruleName, ok := secrets.ScanLine(content); ok {
					cur.SecretLeaks = append(cur.SecretLeaks, fmt.Sprintf("line %d: Potential %s", curLine, ruleName))
				}
				curLine++
			}
			if strings.HasPrefix(line, "+<<<<<<< ") {
				cur.HasConflict = true
			}
		case cur != nil && strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			cur.LinesRemoved++
		case cur != nil && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "\\"):
			if curLine > 0 {
				curLine++
			}
		}
	}
	if cur != nil {
		diffs = append(diffs, *cur)
	}
	// collapse adjacent lines
	for i := range diffs {
		if len(diffs[i].ChangedLines) > 0 {
			var merged [][2]int
			c := diffs[i].ChangedLines[0]
			for j := 1; j < len(diffs[i].ChangedLines); j++ {
				next := diffs[i].ChangedLines[j]
				if next[0] <= c[1]+1 {
					c[1] = next[1]
				} else {
					merged = append(merged, c)
					c = next
				}
			}
			merged = append(merged, c)
			diffs[i].ChangedLines = merged
		}
	}
	return diffs
}
