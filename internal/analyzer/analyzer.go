package analyzer

import (
	"time"

	"github.com/salarkhannn/ghostlog/internal/git"
)

const burstWindow = 5 * time.Second

type Burst struct {
	ID           int
	Hashes       []string
	StartTime    time.Time
	LastTime     time.Time
	LinesAdded   int
	LinesRemoved int
	FilesChanged int
	BytesAdded   int
}

type Analyzer struct {
	bursts     []Burst
	seenHashes map[string]struct{}
	nextID     int
}

func New() *Analyzer {
	return &Analyzer{seenHashes: make(map[string]struct{})}
}

func (a *Analyzer) Add(info *git.CommitInfo) ([]Burst, bool) {
	if len(info.Diffs) == 0 {
		return a.bursts, false
	}
	if _, seen := a.seenHashes[info.Hash]; seen {
		return a.bursts, false
	}
	a.seenHashes[info.Hash] = struct{}{}

	var added, removed, bytesAdded int
	for _, d := range info.Diffs {
		added += d.LinesAdded
		removed += d.LinesRemoved
		bytesAdded += d.Bytes
	}
	files := len(info.Diffs)
	now := time.Now()

	if len(a.bursts) > 0 {
		last := &a.bursts[len(a.bursts)-1]
		if now.Sub(last.LastTime) <= burstWindow {
			last.Hashes = append(last.Hashes, info.Hash)
			last.LastTime = now
			last.LinesAdded += added
			last.LinesRemoved += removed
			last.FilesChanged += files
			last.BytesAdded += bytesAdded
			return a.bursts, true
		}
	}

	a.nextID++
	burst := Burst{
		ID:           a.nextID,
		Hashes:       []string{info.Hash},
		StartTime:    now,
		LastTime:     now,
		LinesAdded:   added,
		LinesRemoved: removed,
		FilesChanged: files,
		BytesAdded:   bytesAdded,
	}

	if len(a.bursts) >= 100 {
		a.bursts = a.bursts[1:]
	}
	a.bursts = append(a.bursts, burst)
	return a.bursts, true
}

func (a *Analyzer) Bursts() []Burst {
	return a.bursts
}
