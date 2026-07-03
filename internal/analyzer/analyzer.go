package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"time"

	"github.com/salarkhannn/ghostlog/internal/git"
)

const burstWindow = 5 * time.Second

type Burst struct {
	ID             int
	Hashes         []string
	StartTime      time.Time
	LastTime       time.Time
	LinesAdded     int
	LinesRemoved   int
	FilesChanged   int
	BytesAdded     int
	HasConflict    bool
	ComplexityBefore int
	ComplexityAfter  int
}

type Analyzer struct {
	repoPath   string
	bursts     []Burst
	seenHashes map[string]struct{}
	nextID     int
}

func New(repoPath string) *Analyzer {
	return &Analyzer{repoPath: repoPath, seenHashes: make(map[string]struct{})}
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
	var compBefore, compAfter int
	hasConflict := false
	for _, d := range info.Diffs {
		added += d.LinesAdded
		removed += d.LinesRemoved
		bytesAdded += d.Bytes
		if d.HasConflict {
			hasConflict = true
		}
		if strings.HasSuffix(d.Path, ".go") {
			beforeCode, _ := git.ShowFile(a.repoPath, info.Hash+"^", d.Path)
			afterCode, _ := git.ShowFile(a.repoPath, info.Hash, d.Path)
			compBefore += calcComplexity(beforeCode)
			compAfter += calcComplexity(afterCode)
		}
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
			last.ComplexityBefore += compBefore
			last.ComplexityAfter += compAfter
			if hasConflict {
				last.HasConflict = true
			}
			return a.bursts, true
		}
	}

	a.nextID++
	burst := Burst{
		ID:               a.nextID,
		Hashes:           []string{info.Hash},
		StartTime:        now,
		LastTime:         now,
		LinesAdded:       added,
		LinesRemoved:     removed,
		FilesChanged:     files,
		BytesAdded:       bytesAdded,
		HasConflict:      hasConflict,
		ComplexityBefore: compBefore,
		ComplexityAfter:  compAfter,
	}

	if len(a.bursts) >= 100 {
		a.bursts = a.bursts[1:]
	}
	a.bursts = append(a.bursts, burst)
	return a.bursts, true
}

func (a *Analyzer) Bursts() []Burst { return a.bursts }

func calcComplexity(src []byte) int {
	if len(src) == 0 {
		return 0
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return 0
	}

	count := 0
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause, *ast.CommClause:
			count++
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				count++
			}
		}
		return true
	})
	return count
}
