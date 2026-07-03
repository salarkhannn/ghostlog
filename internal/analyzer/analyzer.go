package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
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
	UntestedFunctions []string
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
	var untested []string
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

			if !strings.HasSuffix(d.Path, "_test.go") {
				funcs := getChangedFuncs(afterCode, d.ChangedLines)
				if len(funcs) > 0 {
					testFiles, _ := git.ListFiles(a.repoPath, info.Hash, filepath.Dir(d.Path))
					var testCodes [][]byte
					for _, tf := range testFiles {
						if strings.HasSuffix(tf, "_test.go") {
							tc, _ := git.ShowFile(a.repoPath, info.Hash, tf)
							testCodes = append(testCodes, tc)
						}
					}
					for _, fn := range funcs {
						if !isTested(fn, testCodes) {
							untested = append(untested, fn)
						}
					}
				}
			}
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
			if len(untested) > 0 {
				last.UntestedFunctions = append(last.UntestedFunctions, untested...)
			}
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
		UntestedFunctions: untested,
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

func getChangedFuncs(src []byte, changedLines [][2]int) []string {
	if len(src) == 0 || len(changedLines) == 0 {
		return nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil
	}

	var funcs []string
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			start := fset.Position(fn.Pos()).Line
			end := fset.Position(fn.End()).Line
			for _, r := range changedLines {
				if r[0] <= end && r[1] >= start {
					funcs = append(funcs, fn.Name.Name)
					break
				}
			}
		}
	}
	return funcs
}

func isTested(funcName string, testCodes [][]byte) bool {
	for _, src := range testCodes {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, 0)
		if err != nil {
			continue
		}
		found := false
		ast.Inspect(f, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok && ident.Name == funcName {
				found = true
				return false
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}
