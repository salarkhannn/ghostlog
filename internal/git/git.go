package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type FileDiff struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
	RawDiff      string
}

type CommitInfo struct {
	Hash  string
	Diffs []FileDiff
}

func HeadCommit(repoPath string) (string, error) {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func DiffTree(repoPath, commitHash string) (*CommitInfo, error) {
	args := []string{"-C", repoPath, "diff-tree", "--no-commit-id", "-p", "-r", commitHash}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("diff-tree %s: %w", commitHash[:8], err)
	}

	short := commitHash
	if len(short) > 8 {
		short = short[:8]
	}

	info := &CommitInfo{Hash: short}
	info.Diffs = parseDiffOutput(out)
	return info, nil
}

func parseDiffOutput(data []byte) []FileDiff {
	var diffs []FileDiff
	var current *FileDiff
	var rawBuf bytes.Buffer

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				current.RawDiff = rawBuf.String()
				diffs = append(diffs, *current)
			}
			rawBuf.Reset()
			current = &FileDiff{}
			continue
		}

		if strings.HasPrefix(line, "+++ b/") && current != nil {
			current.Path = strings.TrimPrefix(line, "+++ b/")
		}

		if current != nil {
			switch {
			case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
				current.LinesAdded++
			case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
				current.LinesRemoved++
			}
			rawBuf.WriteString(line + "\n")
		}
	}

	if current != nil {
		current.RawDiff = rawBuf.String()
		diffs = append(diffs, *current)
	}

	return diffs
}
