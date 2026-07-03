package log

import (
	"encoding/json"
	"time"

	"github.com/salarkhannn/ghostlog/internal/analyzer"
)

type FileEntry struct {
	Path         string `json:"path"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	IsTest       bool   `json:"is_test"`
	IsGenerated  bool   `json:"is_generated"`
	FuncCount    int    `json:"func_count,omitempty"`
}

type CommitRecord struct {
	Timestamp    time.Time   `json:"timestamp"`
	CommitHash   string      `json:"commit_hash"`
	FilesChanged []FileEntry `json:"files_changed"`
}

func NewRecord(hash string, analyses []analyzer.FileAnalysis) CommitRecord {
	entries := make([]FileEntry, len(analyses))
	for i, a := range analyses {
		entries[i] = FileEntry{
			Path:         a.Path,
			LinesAdded:   a.LinesAdded,
			LinesRemoved: a.LinesRemoved,
			IsTest:       a.IsTest,
			IsGenerated:  a.IsGenerated,
			FuncCount:    a.FuncCount,
		}
	}
	return CommitRecord{
		Timestamp:    time.Now().UTC(),
		CommitHash:   hash,
		FilesChanged: entries,
	}
}

func marshal(r CommitRecord) ([]byte, error) {
	return json.Marshal(r)
}
